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
	Scheme      *runtime.Scheme
	Namespace   string
	GithubToken string
}

//+kubebuilder:rbac:groups=user.openshift.io,resources=groups,verbs=get;list;watch;update;patch
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

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
	reqlog := log.FromContext(ctx)
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

func (r *GroupReconciler) GithubTokenFromSecret(ctx context.Context, group *userv1.Group) (string, error) {
	reqlog := log.FromContext(ctx)

	var githubToken string
	secretName := group.ObjectMeta.Annotations["github.oddbit.com/secret"]
	if len(secretName) > 0 {
		var secret corev1.Secret
		if err := r.Get(ctx, client.ObjectKey{Namespace: r.Namespace, Name: secretName}, &secret); err != nil {
			reqlog.Error(err, "Failed to get secret")
			return "", err
		}

		githubToken = string(secret.Data["GITHUB_TOKEN"])
	}

	if len(githubToken) == 0 {
		githubToken = r.GithubToken

		if len(githubToken) == 0 {
			return "", fmt.Errorf("Missing github token")
		}

		reqlog.V(1).Info("Using global token")
	}

	return githubToken, nil
}

func (r *GroupReconciler) NewGithubClientFromToken(ctx context.Context, group *userv1.Group) (*github.Client, error) {
	githubToken, err := r.GithubTokenFromSecret(ctx, group)
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	tc := oauth2.NewClient(ctx, ts)

	return github.NewClient(tc), nil
}

func (r *GroupReconciler) SyncGroup(ctx context.Context, group *userv1.Group) error {
	reqlog := log.FromContext(ctx)

	orgName := group.ObjectMeta.Annotations["github.oddbit.com/organization"]
	teamName := group.ObjectMeta.Annotations["github.oddbit.com/team"]

	if len(orgName) == 0 {
		return fmt.Errorf("group is missing oddbit.com/organization annotation")
	}

	if len(teamName) == 0 {
		return fmt.Errorf("group is missing oddbit.com/team annotation")
	}

	gh, err := r.NewGithubClientFromToken(ctx, group)
	if err != nil {
		return err
	}

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
