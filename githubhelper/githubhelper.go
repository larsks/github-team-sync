package githubhelper

import (
	"context"

	"github.com/google/go-github/v45/github"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

func GithubTokenFromSecret(ctx context.Context, client client.Client, secretName types.NamespacedName) (string, error) {
	var githubToken string
	var secret corev1.Secret
	if err := client.Get(ctx, secretName, &secret); err != nil {
		return "", err
	}

	githubToken = string(secret.Data["GITHUB_TOKEN"])

	return githubToken, nil
}

func ListTeamMemberNames(ctx context.Context, gh *github.Client, orgName, teamName string) ([]string, error) {
	var members []*github.User

	opts := github.TeamListTeamMembersOptions{
		ListOptions: github.ListOptions{
			PerPage: 30,
		},
	}

	for {
		page, resp, err := gh.Teams.ListTeamMembersBySlug(ctx, orgName, teamName, &opts)
		if err != nil {
			return nil, err
		}
		members = append(members, page...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	memberNames := []string{}
	for _, member := range members {
		memberNames = append(memberNames, member.GetLogin())
	}

	return memberNames, nil
}

func ListTeams(ctx context.Context, gh *github.Client, orgName string) ([]string, error) {
	var teams []*github.Team

	opts := github.ListOptions{
		PerPage: 30,
	}

	for {
		page, resp, err := gh.Teams.ListTeams(ctx, orgName, &opts)
		if err != nil {
			return nil, err
		}
		teams = append(teams, page...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	teamNames := []string{}
	for _, team := range teams {
		teamNames = append(teamNames, team.GetSlug())
	}

	return teamNames, nil
}
