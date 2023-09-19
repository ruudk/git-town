package domain_test

import (
	"testing"

	"github.com/git-town/git-town/v9/src/domain"
	"github.com/stretchr/testify/assert"
)

func TestLocalBranchName(t *testing.T) {
	t.Parallel()
	t.Run("NewLocalBranchName and String", func(t *testing.T) {
		t.Parallel()
		branch := domain.NewLocalBranchName("branch-1")
		assert.Equal(t, "branch-1", branch.String())
	})

	t.Run("IsEmpty", func(t *testing.T) {
		t.Parallel()
		t.Run("branch is empty", func(t *testing.T) {
			t.Parallel()
			branch := domain.LocalBranchName{}
			assert.True(t, branch.IsEmpty())
		})
		t.Run("branch is not empty", func(t *testing.T) {
			t.Parallel()
			branch := domain.NewLocalBranchName("branch")
			assert.False(t, branch.IsEmpty())
		})
	})

	t.Run("RemoteBranch", func(t *testing.T) {
		t.Parallel()
		branch := domain.NewLocalBranchName("branch")
		want := domain.NewRemoteBranchName("origin/branch")
		assert.Equal(t, want, branch.RemoteBranch())
	})
}
