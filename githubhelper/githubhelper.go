package githubhelper

import (
	"context"

	"github.com/google/go-github/v45/github"
)

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
