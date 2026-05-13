package cmdx

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/Olyxz16/tkt/internal/config"
	"github.com/Olyxz16/tkt/internal/local"
	"github.com/Olyxz16/tkt/internal/store"
)

func init() {
	stateCmd.ValidArgsFunction = stateCompletion
	doneCmd.ValidArgsFunction = issueIDCompletion
	editCmd.ValidArgsFunction = issueIDCompletion
	deleteCmd.ValidArgsFunction = issueIDCompletion
	showCmd.ValidArgsFunction = issueIDCompletion
	commentCmd.ValidArgsFunction = issueIDCompletion
	commentsCmd.ValidArgsFunction = issueIDCompletion
	tagAddCmd.ValidArgsFunction = tagAddCompletion
	tagRemoveCmd.ValidArgsFunction = tagRemoveCompletion

	editCmd.RegisterFlagCompletionFunc("state", stateValueCompletion)
	editCmd.RegisterFlagCompletionFunc("priority", priorityValueCompletion)
}

func issueIDCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	var completions []string

	if store.IsInitialized() {
		db, err := store.Open()
		if err == nil {
			defer db.Close()
			issues, err := store.ListIssues(db, "", 100)
			if err == nil {
				for _, issue := range issues {
					if issue.ProviderRef != "" {
						completions = append(completions, issue.ProviderRef)
					}
					completions = append(completions, local.FormatID(&issue))
				}
			}
		}
	}

	return completions, cobra.ShellCompDirectiveNoFileComp
}

func stateCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return issueIDCompletion(cmd, args, toComplete)
	}

	if len(args) == 1 {
		localCfg, _, _ := config.LoadLocal()
		schema := local.GetSchema(localCfg)
		return schema.States, cobra.ShellCompDirectiveNoFileComp
	}

	return nil, cobra.ShellCompDirectiveNoFileComp
}

func tagAddCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return issueIDCompletion(cmd, args, toComplete)
	}
	if len(args) == 1 {
		tags := localTagList()
		if len(tags) > 0 {
			return tags, cobra.ShellCompDirectiveNoFileComp
		}

		svc, _, err := buildService()
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		remoteTags, err := svc.ListTags(context.Background())
		if err != nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var names []string
		for _, t := range remoteTags {
			names = append(names, t.Name)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func tagRemoveCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) == 0 {
		return issueIDCompletion(cmd, args, toComplete)
	}
	if len(args) == 1 {
		tags := localTagList()
		if len(tags) > 0 {
			return tags, cobra.ShellCompDirectiveNoFileComp
		}
	}
	return nil, cobra.ShellCompDirectiveNoFileComp
}

func localTagList() []string {
	if !store.IsInitialized() {
		return nil
	}
	db, err := store.Open()
	if err != nil {
		return nil
	}
	defer db.Close()
	tags, err := store.ListTags(db)
	if err != nil {
		return nil
	}
	return tags
}

func stateValueCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	localCfg, _, _ := config.LoadLocal()
	schema := local.GetSchema(localCfg)
	return schema.States, cobra.ShellCompDirectiveNoFileComp
}

func priorityValueCompletion(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	localCfg, _, _ := config.LoadLocal()
	schema := local.GetSchema(localCfg)
	return schema.Priorities, cobra.ShellCompDirectiveNoFileComp
}