package cmd

import (
	"fmt"

	"github.com/git-town/git-town/v9/src/cli/flags"
	"github.com/git-town/git-town/v9/src/cli/log"
	"github.com/git-town/git-town/v9/src/config"
	"github.com/git-town/git-town/v9/src/domain"
	"github.com/git-town/git-town/v9/src/execute"
	"github.com/git-town/git-town/v9/src/hosting"
	"github.com/git-town/git-town/v9/src/hosting/github"
	"github.com/git-town/git-town/v9/src/messages"
	"github.com/git-town/git-town/v9/src/vm/interpreter"
	"github.com/git-town/git-town/v9/src/vm/persistence"
	"github.com/git-town/git-town/v9/src/vm/runstate"
	"github.com/spf13/cobra"
)

const continueDesc = "Restarts the last run git-town command after having resolved conflicts"

func continueCmd() *cobra.Command {
	addVerboseFlag, readVerboseFlag := flags.Verbose()
	cmd := cobra.Command{
		Use:     "continue",
		GroupID: "errors",
		Args:    cobra.NoArgs,
		Short:   continueDesc,
		Long:    long(continueDesc),
		RunE: func(cmd *cobra.Command, args []string) error {
			return executeContinue(readVerboseFlag(cmd))
		},
	}
	addVerboseFlag(&cmd)
	return &cmd
}

func executeContinue(verbose bool) error {
	repo, err := execute.OpenRepo(execute.OpenRepoArgs{
		Verbose:          verbose,
		DryRun:           false,
		OmitBranchNames:  false,
		ValidateIsOnline: false,
		ValidateGitRepo:  true,
	})
	if err != nil {
		return err
	}
	config, initialBranchesSnapshot, initialStashSnapshot, exit, err := determineContinueConfig(repo, verbose)
	if err != nil || exit {
		return err
	}
	runState, err := determineContinueRunstate(repo)
	if err != nil {
		return err
	}
	return interpreter.Execute(interpreter.ExecuteArgs{
		RunState:                &runState,
		Run:                     &repo.Runner,
		Connector:               config.connector,
		Verbose:                 verbose,
		Lineage:                 config.lineage,
		RootDir:                 repo.RootDir,
		InitialBranchesSnapshot: initialBranchesSnapshot,
		InitialConfigSnapshot:   repo.ConfigSnapshot,
		InitialStashSnapshot:    initialStashSnapshot,
		NoPushHook:              !config.pushHook,
	})
}

func determineContinueConfig(repo *execute.OpenRepoResult, verbose bool) (*continueConfig, domain.BranchesSnapshot, domain.StashSnapshot, bool, error) {
	lineage := repo.Runner.Config.Lineage()
	pushHook, err := repo.Runner.Config.PushHook()
	if err != nil {
		return nil, domain.EmptyBranchesSnapshot(), domain.EmptyStashSnapshot(), false, err
	}
	_, initialBranchesSnapshot, initialStashSnapshot, exit, err := execute.LoadBranches(execute.LoadBranchesArgs{
		Repo:                  repo,
		Verbose:               verbose,
		Fetch:                 false,
		HandleUnfinishedState: false,
		Lineage:               lineage,
		PushHook:              pushHook,
		ValidateIsConfigured:  true,
		ValidateNoOpenChanges: false,
	})
	if err != nil || exit {
		return nil, initialBranchesSnapshot, initialStashSnapshot, exit, err
	}
	repoStatus, err := repo.Runner.Backend.RepoStatus()
	if err != nil {
		return nil, initialBranchesSnapshot, initialStashSnapshot, false, err
	}
	if repoStatus.Conflicts {
		return nil, initialBranchesSnapshot, initialStashSnapshot, false, fmt.Errorf(messages.ContinueUnresolvedConflicts)
	}
	originURL := repo.Runner.Config.OriginURL()
	hostingService, err := repo.Runner.Config.HostingService()
	if err != nil {
		return nil, initialBranchesSnapshot, initialStashSnapshot, false, err
	}
	mainBranch := repo.Runner.Config.MainBranch()
	connector, err := hosting.NewConnector(hosting.NewConnectorArgs{
		HostingService:  hostingService,
		GetSHAForBranch: repo.Runner.Backend.SHAForBranch,
		OriginURL:       originURL,
		GiteaAPIToken:   repo.Runner.Config.GiteaToken(),
		GithubAPIToken:  github.GetAPIToken(repo.Runner.Config),
		GitlabAPIToken:  repo.Runner.Config.GitLabToken(),
		MainBranch:      mainBranch,
		Log:             log.Printing{},
	})
	return &continueConfig{
		connector: connector,
		lineage:   lineage,
		pushHook:  pushHook,
	}, initialBranchesSnapshot, initialStashSnapshot, false, err
}

type continueConfig struct {
	connector hosting.Connector
	lineage   config.Lineage
	pushHook  bool
}

func determineContinueRunstate(repo *execute.OpenRepoResult) (runstate.RunState, error) {
	runState, err := persistence.Load(repo.RootDir)
	if err != nil {
		return runstate.RunState{}, fmt.Errorf(messages.RunstateLoadProblem, err)
	}
	if runState == nil || !runState.IsUnfinished() {
		return runstate.RunState{}, fmt.Errorf(messages.ContinueNothingToDo)
	}
	return *runState, nil
}
