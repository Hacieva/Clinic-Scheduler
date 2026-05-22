package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
)

// mockUserBranchRepo implements repository.UserBranchRepository.
type mockUserBranchRepo struct {
	ids []int64
	has bool
	err error
}

func (m *mockUserBranchRepo) GetBranchIDs(_ context.Context, _ int64) ([]int64, error) {
	return m.ids, m.err
}

func (m *mockUserBranchRepo) SetBranchIDs(_ context.Context, _ int64, _ []int64) error {
	return m.err
}

func (m *mockUserBranchRepo) HasAccess(_ context.Context, _ int64, _ int64) (bool, error) {
	return m.has, m.err
}

// helpers — active/inactive branch constructors (distinct from sampleBranch in branch_test.go)
func activeBranch(id int64, name string) model.Branch {
	return model.Branch{ID: id, Name: name, IsActive: true, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

func inactiveBranch(id int64) model.Branch {
	return model.Branch{ID: id, Name: "inactive", IsActive: false, CreatedAt: time.Now(), UpdatedAt: time.Now()}
}

// — GetBranchIDs —

func TestUserBranchGetBranchIDs_OwnerReturnsAllActive(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{},
		&mockBranchRepo{branches: []model.Branch{
			activeBranch(1, "Branch A"),
			inactiveBranch(2),
			activeBranch(3, "Branch C"),
		}},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 10, model.RoleOwner)
	require.NoError(t, err)
	assert.Equal(t, []int64{1, 3}, ids)
}

func TestUserBranchGetBranchIDs_OwnerNoBranches(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{},
		&mockBranchRepo{branches: []model.Branch{}},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 10, model.RoleOwner)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestUserBranchGetBranchIDs_OwnerOnlyInactive(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{},
		&mockBranchRepo{branches: []model.Branch{inactiveBranch(1), inactiveBranch(2)}},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 10, model.RoleOwner)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestUserBranchGetBranchIDs_AdminReturnsAssigned(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{ids: []int64{2, 5}},
		&mockBranchRepo{},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 20, model.RoleAdmin)
	require.NoError(t, err)
	assert.Equal(t, []int64{2, 5}, ids)
}

func TestUserBranchGetBranchIDs_AdminEmpty(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{ids: []int64{}},
		&mockBranchRepo{},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 20, model.RoleAdmin)
	require.NoError(t, err)
	assert.Empty(t, ids)
}

func TestUserBranchGetBranchIDs_DoctorReturnsAssigned(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{ids: []int64{1}},
		&mockBranchRepo{},
	)

	ids, err := svc.GetBranchIDs(context.Background(), 30, model.RoleDoctor)
	require.NoError(t, err)
	assert.Equal(t, []int64{1}, ids)
}

func TestUserBranchGetBranchIDs_OwnerBranchRepoError(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{},
		&mockBranchRepo{err: apperrors.ErrNotFound},
	)

	_, err := svc.GetBranchIDs(context.Background(), 10, model.RoleOwner)
	assert.Error(t, err)
}

func TestUserBranchGetBranchIDs_AdminRepoError(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{err: apperrors.ErrNotFound},
		&mockBranchRepo{},
	)

	_, err := svc.GetBranchIDs(context.Background(), 20, model.RoleAdmin)
	assert.Error(t, err)
}

// — SetBranchIDs —

func TestUserBranchSetBranchIDs_OwnerForbidden(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 10, model.RoleOwner, []int64{1})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUserBranchSetBranchIDs_AdminSuccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 20, model.RoleAdmin, []int64{1, 2})
	require.NoError(t, err)
}

func TestUserBranchSetBranchIDs_DoctorSuccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 30, model.RoleDoctor, []int64{1})
	require.NoError(t, err)
}

func TestUserBranchSetBranchIDs_AdminEmptyList(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 20, model.RoleAdmin, []int64{})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUserBranchSetBranchIDs_DoctorEmptyList(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 30, model.RoleDoctor, []int64{})
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUserBranchSetBranchIDs_NilListTreatedAsEmpty(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{}, &mockBranchRepo{})

	err := svc.SetBranchIDs(context.Background(), 20, model.RoleAdmin, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestUserBranchSetBranchIDs_RepoError(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{err: apperrors.ErrNotFound},
		&mockBranchRepo{},
	)

	err := svc.SetBranchIDs(context.Background(), 20, model.RoleAdmin, []int64{1})
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — HasAccess —

func TestUserBranchHasAccess_OwnerAlwaysTrue(t *testing.T) {
	// mock returns false — owner must override without calling repo
	svc := NewUserBranchService(&mockUserBranchRepo{has: false}, &mockBranchRepo{})

	ok, err := svc.HasAccess(context.Background(), 10, model.RoleOwner, 99)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUserBranchHasAccess_AdminHasAccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{has: true}, &mockBranchRepo{})

	ok, err := svc.HasAccess(context.Background(), 20, model.RoleAdmin, 1)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUserBranchHasAccess_AdminNoAccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{has: false}, &mockBranchRepo{})

	ok, err := svc.HasAccess(context.Background(), 20, model.RoleAdmin, 99)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestUserBranchHasAccess_DoctorHasAccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{has: true}, &mockBranchRepo{})

	ok, err := svc.HasAccess(context.Background(), 30, model.RoleDoctor, 1)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestUserBranchHasAccess_DoctorNoAccess(t *testing.T) {
	svc := NewUserBranchService(&mockUserBranchRepo{has: false}, &mockBranchRepo{})

	ok, err := svc.HasAccess(context.Background(), 30, model.RoleDoctor, 99)
	require.NoError(t, err)
	assert.False(t, ok)
}

func TestUserBranchHasAccess_RepoError(t *testing.T) {
	svc := NewUserBranchService(
		&mockUserBranchRepo{err: apperrors.ErrNotFound},
		&mockBranchRepo{},
	)

	_, err := svc.HasAccess(context.Background(), 20, model.RoleAdmin, 1)
	assert.Error(t, err)
}
