/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	userv1 "github.com/openshift/api/user/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

// GroupReconciler reconciles a Group object
type GroupReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	Namespace string
}

//+kubebuilder:rbac:groups=user.openshift.io.github.oddbit.com,resources=groups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=user.openshift.io.github.oddbit.com,resources=groups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=user.openshift.io.github.oddbit.com,resources=groups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Group object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	reqlog := log.Log.WithValues("group", req.NamespacedName)
	reqlog.V(1).Info("Triggered")

	var group userv1.Group
	err := r.Get(ctx, req.NamespacedName, &group)
	if err != nil {
		if errors.IsNotFound(err) {
			reqlog.V(1).Info("Group has been deleted")
			return ctrl.Result{}, nil
		} else {
			reqlog.Error(err, "Failed to get group")
			return ctrl.Result{}, err
		}
	}

	selected := group.ObjectMeta.Labels["github.oddbit.com/sync"]
	if selected != "true" {
		reqlog.V(1).Info("Skipping (not labelled)")
		return ctrl.Result{}, nil
	}

	reqlog.Info("Syncing")
	if err := r.SyncGroup(ctx, &group); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GroupReconciler) SyncGroup(ctx context.Context, group *userv1.Group) error {
	reqlog := log.Log.WithValues("group", group.ObjectMeta.Name)

	secretName := group.ObjectMeta.Annotations["github.oddbit.com/secret"]
	orgName := group.ObjectMeta.Annotations["github.oddbit.com/organization"]
	teamName := group.ObjectMeta.Annotations["github.oddbit.com/team"]

	if len(secretName) == 0 {
		return fmt.Errorf("group is missing oddbit.com/secret annotation")
	}

	if len(orgName) == 0 {
		return fmt.Errorf("group is missing oddbit.com/organization annotation")
	}

	if len(teamName) == 0 {
		return fmt.Errorf("group is missing oddbit.com/team annotation")
	}

	var secret corev1.Secret
	if err := r.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: secretName}, &secret); err != nil {
		reqlog.Error(err, "Failed to get secret")
		return err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: string(secret.Data["GITHUB_TOKEN"])},
	)
	tc := oauth2.NewClient(ctx, ts)

	gh := github.NewClient(tc)

	members, _, err := gh.Teams.ListTeamMembersBySlug(ctx, orgName, teamName, nil)
	if err != nil {
		return err
	}

	memberNames := []string{}
	for _, member := range members {
		memberNames = append(memberNames, member.GetLogin())
	}

	if !slices.Equal(memberNames, group.Users) {
		group.Users = memberNames
		reqlog.Info("updating group membership")
		if err := r.Update(ctx, group); err != nil {
			return err
		}
	} else {
		reqlog.Info("no changes to group membership")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&userv1.Group{}).
		Complete(r)
}