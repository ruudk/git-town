package git_test

import (
	"testing"

	"github.com/git-town/git-town/v9/src/config"
	"github.com/git-town/git-town/v9/src/domain"
	"github.com/git-town/git-town/v9/src/git"
	"github.com/git-town/git-town/v9/src/gohacks"
	"github.com/git-town/git-town/v9/src/gohacks/cache"
	"github.com/git-town/git-town/v9/src/subshell"
	testgit "github.com/git-town/git-town/v9/test/git"
	"github.com/git-town/git-town/v9/test/testruntime"
	"github.com/shoenig/test/must"
)

func TestBackendCommands(t *testing.T) {
	t.Parallel()
	initial := domain.NewLocalBranchName("initial")

	t.Run("BranchAuthors", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.Create(t)
		branch := domain.NewLocalBranchName("branch")
		runtime.CreateBranch(branch, initial)
		runtime.CreateCommit(testgit.Commit{
			Branch:      branch,
			FileName:    "file1",
			FileContent: "file1",
			Message:     "first commit",
		})
		runtime.CreateCommit(testgit.Commit{
			Branch:      branch,
			FileName:    "file2",
			FileContent: "file2",
			Message:     "second commit",
		})
		authors, err := runtime.Backend.BranchAuthors(branch, initial)
		must.NoError(t, err)
		must.Eq(t, []string{"user <email@example.com>"}, authors)
	})

	t.Run("BranchHasUnmergedChanges", func(t *testing.T) {
		t.Parallel()
		t.Run("branch without commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			branch := domain.NewLocalBranchName("branch")
			runtime.CreateBranch(branch, initial)
			have, err := runtime.Backend.BranchHasUnmergedChanges(branch, initial)
			must.NoError(t, err)
			must.False(t, have)
		})
		t.Run("branch with commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateCommit(testgit.Commit{
				Branch:      initial,
				Message:     "commit 1",
				FileName:    "file1",
				FileContent: "original content",
			})
			branch := domain.NewLocalBranchName("branch")
			runtime.CreateBranch(branch, initial)
			runtime.CreateCommit(testgit.Commit{
				Branch:      branch,
				Message:     "commit 2",
				FileName:    "file1",
				FileContent: "modified content",
			})
			have, err := runtime.Backend.BranchHasUnmergedChanges(branch, initial)
			must.NoError(t, err)
			must.True(t, have, must.Sprint("branch with commits that make changes"))
			runtime.CreateCommit(testgit.Commit{
				Branch:      branch,
				Message:     "commit 3",
				FileName:    "file1",
				FileContent: "original content",
			})
			have, err = runtime.Backend.BranchHasUnmergedChanges(branch, initial)
			must.NoError(t, err)
			must.False(t, have, must.Sprint("branch with commits that make no changes"))
		})
	})

	t.Run("BranchHasUnmergedCommits", func(t *testing.T) {
		t.Parallel()
		t.Run("branch without commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			branch := domain.NewLocalBranchName("branch")
			runtime.CreateBranch(branch, initial)
			have, err := runtime.Backend.BranchHasUnmergedCommits(branch, initial.Location())
			must.NoError(t, err)
			must.False(t, have)
		})
		t.Run("branch with commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateCommit(testgit.Commit{
				Branch:      initial,
				Message:     "commit 1",
				FileName:    "file1",
				FileContent: "original content",
			})
			branch := domain.NewLocalBranchName("branch")
			runtime.CreateBranch(branch, initial)
			runtime.CreateCommit(testgit.Commit{
				Branch:      branch,
				Message:     "commit 2",
				FileName:    "file1",
				FileContent: "modified content",
			})
			have, err := runtime.Backend.BranchHasUnmergedCommits(branch, initial.Location())
			must.NoError(t, err)
			must.True(t, have, must.Sprint("branch with commits that make changes"))
			runtime.CreateCommit(testgit.Commit{
				Branch:      branch,
				Message:     "commit 3",
				FileName:    "file1",
				FileContent: "original content",
			})
			have, err = runtime.Backend.BranchHasUnmergedCommits(branch, initial.Location())
			must.NoError(t, err)
			must.True(t, have, must.Sprint("branch with commits that make no changes"))
		})
	})

	t.Run("CheckoutBranch", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.Create(t)
		runtime.CreateBranch(domain.NewLocalBranchName("branch1"), initial)
		must.NoError(t, runtime.Backend.CheckoutBranch(domain.NewLocalBranchName("branch1")))
		currentBranch, err := runtime.CurrentBranch()
		must.NoError(t, err)
		must.EqOp(t, domain.NewLocalBranchName("branch1"), currentBranch)
		runtime.CheckoutBranch(initial)
		currentBranch, err = runtime.CurrentBranch()
		must.NoError(t, err)
		must.EqOp(t, initial, currentBranch)
	})

	t.Run("CommitsInBranch", func(t *testing.T) {
		t.Parallel()
		t.Run("feature branch contains commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateBranch(domain.NewLocalBranchName("branch1"), initial)
			runtime.CreateCommit(testgit.Commit{
				Branch:   domain.NewLocalBranchName("branch1"),
				Message:  "commit 1",
				FileName: "file1",
			})
			runtime.CreateCommit(testgit.Commit{
				Branch:   domain.NewLocalBranchName("branch1"),
				Message:  "commit 2",
				FileName: "file2",
			})
			commits, err := runtime.BackendCommands.CommitsInBranch(domain.NewLocalBranchName("branch1"), domain.NewLocalBranchName("initial"))
			must.NoError(t, err)
			must.EqOp(t, 2, len(commits))
		})
		t.Run("feature branch contains no commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateBranch(domain.NewLocalBranchName("branch1"), initial)
			commits, err := runtime.BackendCommands.CommitsInBranch(domain.NewLocalBranchName("branch1"), domain.NewLocalBranchName("initial"))
			must.NoError(t, err)
			must.EqOp(t, 0, len(commits))
		})
		t.Run("main branch contains commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateCommit(testgit.Commit{
				Branch:   initial,
				Message:  "commit 1",
				FileName: "file1",
			})
			runtime.CreateCommit(testgit.Commit{
				Branch:   initial,
				Message:  "commit 2",
				FileName: "file2",
			})
			commits, err := runtime.BackendCommands.CommitsInBranch(domain.NewLocalBranchName("initial"), domain.EmptyLocalBranchName())
			must.NoError(t, err)
			must.EqOp(t, 3, len(commits)) // 1 initial commit + 2 test commits
		})
		t.Run("main branch contains no commits", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			commits, err := runtime.BackendCommands.CommitsInBranch(domain.NewLocalBranchName("initial"), domain.EmptyLocalBranchName())
			must.NoError(t, err)
			must.EqOp(t, 1, len(commits)) // the initial commit
		})
	})

	t.Run("CreateFeatureBranch", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.CreateGitTown(t)
		err := runtime.Backend.CreateFeatureBranch(domain.NewLocalBranchName("f1"))
		must.NoError(t, err)
		runtime.Config.Reload()
		must.True(t, runtime.Config.BranchTypes().IsFeatureBranch(domain.NewLocalBranchName("f1")))
		lineageHave := runtime.Config.Lineage()
		lineageWant := config.Lineage{}
		lineageWant[domain.NewLocalBranchName("f1")] = domain.NewLocalBranchName("main")
		must.Eq(t, lineageWant, lineageHave)
	})

	t.Run("CurrentBranch", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.Create(t)
		runtime.CheckoutBranch(initial)
		runtime.CreateBranch(domain.NewLocalBranchName("b1"), initial)
		runtime.CheckoutBranch(domain.NewLocalBranchName("b1"))
		branch, err := runtime.Backend.CurrentBranch()
		must.NoError(t, err)
		must.EqOp(t, domain.NewLocalBranchName("b1"), branch)
		runtime.CheckoutBranch(initial)
		branch, err = runtime.Backend.CurrentBranch()
		must.NoError(t, err)
		must.EqOp(t, initial, branch)
	})

	t.Run("FirstExistingBranch", func(t *testing.T) {
		t.Parallel()
		t.Run("first branch matches", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			branch1 := domain.NewLocalBranchName("b1")
			branch2 := domain.NewLocalBranchName("b2")
			runtime.CreateBranch(branch1, initial)
			runtime.CreateBranch(branch2, initial)
			branchNames := domain.LocalBranchNames{branch1, branch2}
			have := runtime.Backend.FirstExistingBranch(branchNames, domain.NewLocalBranchName("main"))
			want := branch1
			must.EqOp(t, want, have)
		})
		t.Run("second branch matches", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			branch1 := domain.NewLocalBranchName("b1")
			branch2 := domain.NewLocalBranchName("b2")
			runtime.CreateBranch(branch2, initial)
			branchNames := domain.LocalBranchNames{branch1, branch2}
			have := runtime.Backend.FirstExistingBranch(branchNames, domain.NewLocalBranchName("main"))
			want := branch2
			must.EqOp(t, want, have)
		})
		t.Run("no branch matches", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			branch1 := domain.NewLocalBranchName("b1")
			branch2 := domain.NewLocalBranchName("b2")
			main := domain.NewLocalBranchName("main")
			branchNames := domain.LocalBranchNames{branch1, branch2}
			have := runtime.Backend.FirstExistingBranch(branchNames, main)
			want := main
			must.EqOp(t, want, have)
		})
	})

	t.Run("HasLocalBranch", func(t *testing.T) {
		t.Parallel()
		origin := testruntime.Create(t)
		repoDir := t.TempDir()
		runner := testruntime.Clone(origin.TestRunner, repoDir)
		runner.CreateBranch(domain.NewLocalBranchName("b1"), initial)
		runner.CreateBranch(domain.NewLocalBranchName("b2"), initial)
		must.True(t, runner.Backend.HasLocalBranch(domain.NewLocalBranchName("b1")))
		must.True(t, runner.Backend.HasLocalBranch(domain.NewLocalBranchName("b2")))
		must.False(t, runner.Backend.HasLocalBranch(domain.NewLocalBranchName("b3")))
	})

	t.Run("RepoStatus", func(t *testing.T) {
		t.Run("HasOpenChanges", func(t *testing.T) {
			t.Parallel()
			t.Run("no open changes", func(t *testing.T) {
				t.Parallel()
				runtime := testruntime.Create(t)
				have, err := runtime.Backend.RepoStatus()
				must.NoError(t, err)
				must.False(t, have.OpenChanges)
			})
			t.Run("has open changes", func(t *testing.T) {
				t.Parallel()
				runtime := testruntime.Create(t)
				runtime.CreateFile("foo", "bar")
				have, err := runtime.Backend.RepoStatus()
				must.NoError(t, err)
				must.True(t, have.OpenChanges)
			})
			t.Run("during rebase", func(t *testing.T) {
				t.Parallel()
				runtime := testruntime.Create(t)
				branch1 := domain.NewLocalBranchName("branch1")
				runtime.CreateBranch(branch1, initial)
				runtime.CheckoutBranch(branch1)
				runtime.CreateCommit(testgit.Commit{
					Branch:      branch1,
					FileName:    "file",
					FileContent: "content on branch1",
					Message:     "Create file",
				})
				runtime.CheckoutBranch(initial)
				runtime.CreateCommit(testgit.Commit{
					Branch:      initial,
					FileName:    "file",
					FileContent: "content on initial",
					Message:     "Create file1",
				})
				_ = runtime.RebaseAgainstBranch(branch1) // this is expected to fail
				have, err := runtime.Backend.RepoStatus()
				must.NoError(t, err)
				must.False(t, have.OpenChanges)
			})
			t.Run("during merge conflict", func(t *testing.T) {
				t.Parallel()
				runtime := testruntime.Create(t)
				branch1 := domain.NewLocalBranchName("branch1")
				runtime.CreateBranch(branch1, initial)
				runtime.CheckoutBranch(branch1)
				runtime.CreateCommit(testgit.Commit{
					Branch:      branch1,
					FileName:    "file",
					FileContent: "content on branch1",
					Message:     "Create file",
				})
				runtime.CheckoutBranch(initial)
				runtime.CreateCommit(testgit.Commit{
					Branch:      initial,
					FileName:    "file",
					FileContent: "content on initial",
					Message:     "Create file1",
				})
				_ = runtime.MergeBranch(branch1) // this is expected to fail
				have, err := runtime.Backend.RepoStatus()
				must.NoError(t, err)
				must.False(t, have.OpenChanges)
			})
			t.Run("unstashed conflicting changes", func(t *testing.T) {
				t.Parallel()
				runtime := testruntime.Create(t)
				runtime.CreateFile("file", "stashed content")
				runtime.StashOpenFiles()
				runtime.CreateCommit(testgit.Commit{
					Branch:      initial,
					FileName:    "file",
					FileContent: "committed content",
					Message:     "Create file",
				})
				_ = runtime.UnstashOpenFiles() // this is expected to fail
				have, err := runtime.Backend.RepoStatus()
				must.NoError(t, err)
				must.True(t, have.OpenChanges)
			})
		})

		t.Run("RebaseInProgress", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			have, err := runtime.Backend.RepoStatus()
			must.NoError(t, err)
			must.False(t, have.RebaseInProgress)
		})
	})

	t.Run("ParseVerboseBranchesOutput", func(t *testing.T) {
		t.Parallel()
		t.Run("recognizes the current branch", func(t *testing.T) {
			t.Parallel()
			t.Run("marker is at the first entry", func(t *testing.T) {
				t.Parallel()
				give := `
* branch-1                     01a7eded [origin/branch-1: ahead 1] Commit message 1
  branch-2                     da796a69 [origin/branch-2] Commit message 2
  branch-3                     f4ebec0a [origin/branch-3: behind 2] Commit message 3a`[1:]
				_, currentBranch := git.ParseVerboseBranchesOutput(give)
				must.EqOp(t, domain.NewLocalBranchName("branch-1"), currentBranch)
			})
			t.Run("marker is at the middle entry", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     01a7eded [origin/branch-1: ahead 1] Commit message 1
* branch-2                     da796a69 [origin/branch-2] Commit message 2
  branch-3                     f4ebec0a [origin/branch-3: behind 2] Commit message 3a`[1:]
				_, currentBranch := git.ParseVerboseBranchesOutput(give)
				must.EqOp(t, domain.NewLocalBranchName("branch-2"), currentBranch)
			})
			t.Run("marker is at the last entry", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     01a7eded [origin/branch-1: ahead 1] Commit message 1
  branch-2                     da796a69 [origin/branch-2] Commit message 2
* branch-3                     f4ebec0a [origin/branch-3: behind 2] Commit message 3a`[1:]
				_, currentBranch := git.ParseVerboseBranchesOutput(give)
				must.EqOp(t, domain.NewLocalBranchName("branch-3"), currentBranch)
			})
		})

		t.Run("recognizes the branch sync status", func(t *testing.T) {
			t.Parallel()
			t.Run("branch is ahead of its remote branch", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     111111 [origin/branch-1: ahead 1] Commit message 1a
  remotes/origin/branch-1      222222 Commit message 1b`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("111111"),
						SyncStatus: domain.SyncStatusAhead,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("222222"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("branch is behind its remote branch", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     111111 [origin/branch-1: behind 2] Commit message 1
  remotes/origin/branch-1      222222 Commit message 1b`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("111111"),
						SyncStatus: domain.SyncStatusBehind,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("222222"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("branch is ahead and behind its remote branch", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     111111 [origin/branch-1: ahead 31, behind 2] Commit message 1a
  remotes/origin/branch-1      222222 Commit message 1b`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("111111"),
						SyncStatus: domain.SyncStatusAheadAndBehind,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("222222"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("branch is in sync with its remote branch", func(t *testing.T) {
				t.Parallel()
				give := `
  branch-1                     111111 [origin/branch-1] Commit message 1
  remotes/origin/branch-1      111111 Commit message 1`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("111111"),
						SyncStatus: domain.SyncStatusUpToDate,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("111111"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("remote-only branch", func(t *testing.T) {
				t.Parallel()
				give := `
  remotes/origin/branch-1    222222 Commit message 2`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.EmptyLocalBranchName(),
						LocalSHA:   domain.EmptySHA(),
						SyncStatus: domain.SyncStatusRemoteOnly,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("222222"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("local-only branch", func(t *testing.T) {
				t.Parallel()
				give := `  branch-1                     01a7eded Commit message 1`
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("01a7eded"),
						SyncStatus: domain.SyncStatusLocalOnly,
						RemoteName: domain.EmptyRemoteBranchName(),
						RemoteSHA:  domain.EmptySHA(),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})

			t.Run("branch is deleted at the remote", func(t *testing.T) {
				t.Parallel()
				give := `  branch-1                     01a7eded [origin/branch-1: gone] Commit message 1`
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("01a7eded"),
						SyncStatus: domain.SyncStatusDeletedAtRemote,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.EmptySHA(),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})
		})

		t.Run("branch with a different tracking branch name", func(t *testing.T) {
			t.Run("a branch uses a differently named tracking branch", func(t *testing.T) {
				give := `
  branch-1                     111111 [origin/branch-2] Commit message 1
  remotes/origin/branch-1      222222 Commit message 2
  remotes/origin/branch-2      111111 Commit message 1`[1:]
				want := domain.BranchInfos{
					domain.BranchInfo{
						LocalName:  domain.NewLocalBranchName("branch-1"),
						LocalSHA:   domain.NewSHA("111111"),
						SyncStatus: domain.SyncStatusUpToDate,
						RemoteName: domain.NewRemoteBranchName("origin/branch-2"),
						RemoteSHA:  domain.NewSHA("111111"),
					},
					domain.BranchInfo{
						LocalName:  domain.EmptyLocalBranchName(),
						LocalSHA:   domain.EmptySHA(),
						SyncStatus: domain.SyncStatusRemoteOnly,
						RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
						RemoteSHA:  domain.NewSHA("222222"),
					},
				}
				have, _ := git.ParseVerboseBranchesOutput(give)
				must.Eq(t, want, have)
			})
		})

		t.Run("complex example", func(t *testing.T) {
			give := `
  branch-1                     01a7eded [origin/branch-1: ahead 1] Commit message 1a
* branch-2                     da796a69 [origin/branch-2] Commit message 2
  branch-3                     f4ebec0a [origin/branch-3: behind 2] Commit message 3a
  main                         024df944 [origin/main] Commit message on main (#1234)
  branch-4                     e4d6bc09 [origin/branch-4: gone] Commit message 4
  remotes/origin/branch-1      307a7bf4 Commit message 1b
  remotes/origin/branch-2      da796a69 Commit message 2
  remotes/origin/branch-3      bc39378a Commit message 3b
  remotes/origin/HEAD          -> origin/initial
  remotes/origin/main          024df944 Commit message on main (#1234)
`[1:]
			want := domain.BranchInfos{
				domain.BranchInfo{
					LocalName:  domain.NewLocalBranchName("branch-1"),
					LocalSHA:   domain.NewSHA("01a7eded"),
					SyncStatus: domain.SyncStatusAhead,
					RemoteName: domain.NewRemoteBranchName("origin/branch-1"),
					RemoteSHA:  domain.NewSHA("307a7bf4"),
				},
				domain.BranchInfo{
					LocalName:  domain.NewLocalBranchName("branch-2"),
					LocalSHA:   domain.NewSHA("da796a69"),
					SyncStatus: domain.SyncStatusUpToDate,
					RemoteName: domain.NewRemoteBranchName("origin/branch-2"),
					RemoteSHA:  domain.NewSHA("da796a69"),
				},
				domain.BranchInfo{
					LocalName:  domain.NewLocalBranchName("branch-3"),
					LocalSHA:   domain.NewSHA("f4ebec0a"),
					SyncStatus: domain.SyncStatusBehind,
					RemoteName: domain.NewRemoteBranchName("origin/branch-3"),
					RemoteSHA:  domain.NewSHA("bc39378a"),
				},
				domain.BranchInfo{
					LocalName:  domain.NewLocalBranchName("main"),
					LocalSHA:   domain.NewSHA("024df944"),
					SyncStatus: domain.SyncStatusUpToDate,
					RemoteName: domain.NewRemoteBranchName("origin/main"),
					RemoteSHA:  domain.NewSHA("024df944"),
				},
				domain.BranchInfo{
					LocalName:  domain.NewLocalBranchName("branch-4"),
					LocalSHA:   domain.NewSHA("e4d6bc09"),
					SyncStatus: domain.SyncStatusDeletedAtRemote,
					RemoteName: domain.NewRemoteBranchName("origin/branch-4"),
					RemoteSHA:  domain.EmptySHA(),
				},
			}
			have, currentBranch := git.ParseVerboseBranchesOutput(give)
			must.Eq(t, want, have)
			must.EqOp(t, domain.NewLocalBranchName("branch-2"), currentBranch)
		})
	})

	t.Run("PreviouslyCheckedOutBranch", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.Create(t)
		runtime.CreateBranch(domain.NewLocalBranchName("feature1"), initial)
		runtime.CreateBranch(domain.NewLocalBranchName("feature2"), initial)
		runtime.CheckoutBranch(domain.NewLocalBranchName("feature1"))
		runtime.CheckoutBranch(domain.NewLocalBranchName("feature2"))
		have := runtime.Backend.PreviouslyCheckedOutBranch()
		must.EqOp(t, domain.NewLocalBranchName("feature1"), have)
	})

	t.Run("Remotes", func(t *testing.T) {
		t.Parallel()
		runtime := testruntime.Create(t)
		origin := testruntime.Create(t)
		runtime.AddRemote(domain.OriginRemote, origin.WorkingDir)
		remotes, err := runtime.Backend.Remotes()
		must.NoError(t, err)
		must.Eq(t, domain.Remotes{domain.OriginRemote}, remotes)
	})

	t.Run("RootDirectory", func(t *testing.T) {
		t.Parallel()
		t.Run("inside a Git repo", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			have := runtime.BackendCommands.RootDirectory()
			must.False(t, have.IsEmpty())
		})
		t.Run("outside a Git repo", func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			runner := subshell.BackendRunner{
				Dir:             &dir,
				Verbose:         false,
				CommandsCounter: &gohacks.Counter{},
			}
			cmds := git.BackendCommands{
				BackendRunner:      runner,
				Config:             nil,
				CurrentBranchCache: &cache.LocalBranch{},
				RemotesCache:       &cache.Remotes{},
			}
			have := cmds.RootDirectory()
			want := domain.EmptyRepoRootDir()
			must.EqOp(t, want, have)
		})
	})

	t.Run("StashEntries", func(t *testing.T) {
		t.Parallel()
		t.Run("some stash entries", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			runtime.CreateFile("file1", "content")
			runtime.StashOpenFiles()
			runtime.CreateFile("file2", "content")
			runtime.StashOpenFiles()
			have, err := runtime.StashSnapshot()
			want := domain.StashSnapshot{
				Amount: 2,
			}
			must.NoError(t, err)
			must.EqOp(t, want, have)
		})
		t.Run("no stash entries", func(t *testing.T) {
			t.Parallel()
			runtime := testruntime.Create(t)
			have, err := runtime.StashSnapshot()
			want := domain.StashSnapshot{
				Amount: 0,
			}
			must.NoError(t, err)
			must.EqOp(t, want, have)
		})
	})
}
