/*
Copyright 2022 Lars Kellogg-Stedman.

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
	"sort"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/strings/slices"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/google/go-github/v45/github"
	githubv1alpha1 "github.com/larsks/github-team-sync/api/v1alpha1"
	"github.com/larsks/github-team-sync/githubhelper"
	userv1 "github.com/openshift/api/user/v1"
	"golang.org/x/exp/maps"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// GroupSyncReconciler reconciles a GroupSync object
type GroupSyncReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=github.oddbit.com,resources=groupsyncs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=github.oddbit.com,resources=groupsyncs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=github.oddbit.com,resources=groupsyncs/finalizers,verbs=update
//+kubebuilder:rbac:groups=user.openshift.io,resources=Group,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups=core,resources=Secret,verbs=get

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.12.1/pkg/reconcile
func (r *GroupSyncReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqlog := log.FromContext(ctx)

	reqlog.Info("start reconcile loop")

	var groupsync githubv1alpha1.GroupSync
	if err := r.Get(ctx, req.NamespacedName, &groupsync); err != nil {
		if errors.IsNotFound(err) {
			reqlog.Info("GroupSync resource has been deleted")
			return ctrl.Result{}, nil
		} else {
			return ctrl.Result{}, err
		}
	}

	if err := r.SyncTeams(ctx, &groupsync); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *GroupSyncReconciler) SyncTeams(ctx context.Context, gs *githubv1alpha1.GroupSync) error {
	reqlog := log.FromContext(ctx)

	ght, err := githubhelper.GithubTokenFromSecret(ctx, r.Client, gs.Spec.GithubTokenSecret.AsNamespacedName())
	if err != nil {
		return err
	}

	gh, err := githubhelper.NewGithubClient(ctx, ght)
	if err != nil {
		return err
	}

	teams, err := GetTeamsToSync(ctx, gh, gs)
	if err != nil {
		return err
	}

	for _, teamName := range teams {
		groupName := gs.Spec.Teams[teamName]
		if len(groupName) == 0 {
			groupName = teamName
		}

		reqlog := reqlog.WithValues("team", teamName, "group", groupName)

		members, err := githubhelper.ListTeamMemberNames(ctx, gh, gs.Spec.Organization, teamName)
		if err != nil {
			return err
		}
		reqlog.WithValues("members", members).Info("found members for team")

		if err := r.SetGroupMembership(ctx, gs, groupName, members); err != nil {
			if errors.IsNotFound(err) {
				reqlog.Info("group not found (ignoring)")
				continue
			} else {
				return err
			}
		}
	}

	return nil
}

func (r *GroupSyncReconciler) SetGroupMembership(ctx context.Context, gs *githubv1alpha1.GroupSync, groupName string, members []string) error {
	reqlog := log.FromContext(ctx).WithValues("group", groupName)
	reqlog.Info("reconciling group membership")

	var group userv1.Group
	var createGroup bool

	if err := r.Get(ctx, types.NamespacedName{Name: groupName}, &group); err != nil {
		if errors.IsNotFound(err) {
			createGroup = true
			group = userv1.Group{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Group",
					APIVersion: "user.openshift.io/v1",
				},
			}
			group.Name = groupName
		} else {
			return err
		}
	}

	if err := ctrl.SetControllerReference(gs, &group, r.Scheme); err != nil {
		return err
	}

	if !EqualIgnoringOrder(members, group.Users) {
		reqlog.Info("updating group membership")
		group.Users = members
		var err error

		if createGroup {
			err = r.Create(ctx, &group)
		} else {
			err = r.Update(ctx, &group)
		}

		if err != nil {
			return err
		}
	} else {
		reqlog.Info("no changes to group membership")
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *GroupSyncReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&githubv1alpha1.GroupSync{}).
		Owns(&userv1.Group{}).
		Complete(r)
}

func GetTeamsToSync(ctx context.Context, gh *github.Client, gs *githubv1alpha1.GroupSync) ([]string, error) {
	var teams []string
	var err error
	if gs.Spec.SyncAllTeams {
		teams, err = githubhelper.ListTeams(ctx, gh, gs.Spec.Organization)
		if err != nil {
			return nil, err
		}
	} else {
		teams = maps.Keys(gs.Spec.Teams)
	}

	return teams, nil
}

// Compare two string slices, returning True if they both contain the same
// items, regardless of order.
func EqualIgnoringOrder(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	sort.Strings(a)
	sort.Strings(b)

	return slices.Equal(a, b)
}
