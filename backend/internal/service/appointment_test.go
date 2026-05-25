package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// mockAppointmentRepo implements repository.AppointmentRepository for service-layer tests.
type mockAppointmentRepo struct {
	appt      *model.Appointment
	detail    *repository.AppointmentDetail
	list      []repository.AppointmentDetail
	err       error
	updateErr error // separate error for UpdateStatus
}

func (m *mockAppointmentRepo) Create(_ context.Context, _ repository.CreateAppointmentInput) (*model.Appointment, error) {
	return m.appt, m.err
}

func (m *mockAppointmentRepo) GetByID(_ context.Context, _ int64) (*repository.AppointmentDetail, error) {
	return m.detail, m.err
}

func (m *mockAppointmentRepo) List(_ context.Context, _ repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return m.list, m.err
}

func (m *mockAppointmentRepo) UpdateStatus(_ context.Context, _ int64, _, _ model.AppointmentStatus, _ *int64, _ *string) error {
	return m.updateErr
}

// mockDoctorServiceRepo implements repository.DoctorServiceRepository for service-layer tests.
type mockDoctorServiceRepo struct {
	assigned bool
	err      error
}

func (m *mockDoctorServiceRepo) ListAssignedToDoctor(_ context.Context, _ int64) ([]model.Service, error) {
	return nil, m.err
}

func (m *mockDoctorServiceRepo) IsAssigned(_ context.Context, _, _ int64) (bool, error) {
	return m.assigned, m.err
}

func (m *mockDoctorServiceRepo) Assign(_ context.Context, _, _ int64) error      { return m.err }
func (m *mockDoctorServiceRepo) Unassign(_ context.Context, _, _ int64) error    { return m.err }
func (m *mockDoctorServiceRepo) BulkReplace(_ context.Context, _ int64, _ []int64) error {
	return m.err
}

// helpers

func sampleAppt() *model.Appointment {
	return &model.Appointment{
		ID:        1,
		DoctorID:  1,
		ServiceID: 1,
		Status:    model.StatusCreated,
		StartAt:   time.Now().Add(2 * time.Hour),
		EndAt:     time.Now().Add(3 * time.Hour),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func sampleApptDetail(status model.AppointmentStatus) *repository.AppointmentDetail {
	return &repository.AppointmentDetail{
		Appointment: model.Appointment{
			ID:        1,
			DoctorID:  1,
			ServiceID: 1,
			Status:    status,
			StartAt:   time.Now().Add(2 * time.Hour),
			EndAt:     time.Now().Add(3 * time.Hour),
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		PatientName:    "Ivan Ivanov",
		PatientPhone:   "+79001234567",
		DoctorFullName: "John Smith",
		ServiceName:    "Consultation",
	}
}

func int64Ptr(v int64) *int64 { return &v }

func activeSvc() *model.Service {
	return &model.Service{
		ID:              1,
		DoctorID:        int64Ptr(1), // TODO: legacy field; assignment validated via doctorSvcRepo
		DirectionID:     int64Ptr(1),
		Name:            "Consultation",
		DurationMinutes: 30,
		IsActive:        true,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}
}

func sampleCreateInput() CreateAppointmentInput {
	return CreateAppointmentInput{
		PatientName:  "Ivan Ivanov",
		PatientPhone: "+79001234567",
		DoctorID:     1,
		ServiceID:    1,
		StartAt:      time.Now().Add(2 * time.Hour),
		Source:       model.SourceAdminPanel,
	}
}

// newApptSvc builds an AppointmentService with a default doctorSvcRepo that
// reports the service as assigned (IsAssigned = true). Tests that need to
// override the assignment mock should construct AppointmentService directly.
func newApptSvc(apptRepo *mockAppointmentRepo, docRepo *mockDoctorRepo, svcRepo *mockServiceRepo) *AppointmentService {
	return NewAppointmentService(apptRepo, docRepo, svcRepo, &mockDoctorServiceRepo{assigned: true})
}

// — Create —

func TestAppointmentCreate_Success(t *testing.T) {
	doc := sampleDoctorWithDir()
	svc := newApptSvc(
		&mockAppointmentRepo{appt: sampleAppt()},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, model.StatusCreated, result.Status)
}

func TestAppointmentCreate_DoctorNotFound(t *testing.T) {
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{err: apperrors.ErrNotFound},
		&mockServiceRepo{},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestAppointmentCreate_DoctorInactive(t *testing.T) {
	doc := &model.DoctorWithDirections{
		Doctor: model.Doctor{
			ID: 1, IsActive: false, CreatedAt: time.Now(), UpdatedAt: time.Now(),
		},
		Directions: []model.Direction{},
	}
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrDoctorInactive)
	assert.Nil(t, result)
}

func TestAppointmentCreate_ServiceNotFound(t *testing.T) {
	doc := sampleDoctorWithDir()
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{err: apperrors.ErrNotFound},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestAppointmentCreate_ServiceInactive(t *testing.T) {
	doc := sampleDoctorWithDir()
	inactive := activeSvc()
	inactive.IsActive = false
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: inactive},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

func TestAppointmentCreate_ServiceWrongDoctor(t *testing.T) {
	doc := sampleDoctorWithDir()
	// Service exists and is active, but doctor is NOT assigned to it via junction.
	svc := NewAppointmentService(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: false},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrDirectionMismatch)
	assert.Nil(t, result)
}

func TestAppointmentCreate_StartInPast(t *testing.T) {
	doc := sampleDoctorWithDir()
	input := sampleCreateInput()
	input.StartAt = time.Now().Add(-1 * time.Hour)
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	result, err := svc.Create(context.Background(), input)
	assert.ErrorIs(t, err, apperrors.ErrOutsideHours)
	assert.Nil(t, result)
}

func TestAppointmentCreate_SlotTaken(t *testing.T) {
	doc := sampleDoctorWithDir()
	svc := newApptSvc(
		&mockAppointmentRepo{err: apperrors.ErrSlotTaken},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrSlotTaken)
	assert.Nil(t, result)
}

func TestAppointmentCreate_EndAtComputedFromDuration(t *testing.T) {
	doc := sampleDoctorWithDir()
	medSvc := activeSvc()
	medSvc.DurationMinutes = 45
	start := time.Now().Add(3 * time.Hour).Truncate(time.Minute)
	svc := newApptSvc(
		&mockAppointmentRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: medSvc},
	)
	input := sampleCreateInput()
	input.StartAt = start

	_, err := svc.Create(context.Background(), input)
	require.NoError(t, err)
}

// — GetByID —

func TestAppointmentGetByID_Success(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	result, err := svc.GetByID(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, int64(1), result.ID)
}

func TestAppointmentGetByID_NotFound(t *testing.T) {
	svc := newApptSvc(
		&mockAppointmentRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	result, err := svc.GetByID(context.Background(), 999)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
	assert.Nil(t, result)
}

// — List —

func TestAppointmentList_Success(t *testing.T) {
	details := []repository.AppointmentDetail{*sampleApptDetail(model.StatusCreated)}
	svc := newApptSvc(
		&mockAppointmentRepo{list: details},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	result, err := svc.List(context.Background(), repository.AppointmentFilter{})
	require.NoError(t, err)
	assert.Len(t, result, 1)
}

func TestAppointmentList_Empty(t *testing.T) {
	svc := newApptSvc(
		&mockAppointmentRepo{list: []repository.AppointmentDetail{}},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	result, err := svc.List(context.Background(), repository.AppointmentFilter{})
	require.NoError(t, err)
	assert.Empty(t, result)
}

// — Confirm —

func TestAppointmentConfirm_Success(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Confirm(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestAppointmentConfirm_InvalidTransition(t *testing.T) {
	detail := sampleApptDetail(model.StatusCompleted)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Confirm(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// — CancelByAdmin —

func TestAppointmentCancelByAdmin_FromCreated(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.CancelByAdmin(context.Background(), 1, nil, nil)
	require.NoError(t, err)
}

func TestAppointmentCancelByAdmin_FromConfirmed(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.CancelByAdmin(context.Background(), 1, nil, nil)
	require.NoError(t, err)
}

func TestAppointmentCancelByAdmin_FromCompleted(t *testing.T) {
	detail := sampleApptDetail(model.StatusCompleted)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.CancelByAdmin(context.Background(), 1, nil, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// — CancelByPatient —

func TestAppointmentCancelByPatient_FromCreated(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.CancelByPatient(context.Background(), 1)
	require.NoError(t, err)
}

func TestAppointmentCancelByPatient_FromConfirmed(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.CancelByPatient(context.Background(), 1)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// — Complete —

func TestAppointmentComplete_FromConfirmed(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Complete(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestAppointmentComplete_FromCreated(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Complete(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// — MarkNoShow —

func TestAppointmentMarkNoShow_FromConfirmed(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.MarkNoShow(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestAppointmentMarkNoShow_FromCreated(t *testing.T) {
	detail := sampleApptDetail(model.StatusCreated)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.MarkNoShow(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// — Status machine: not found —

func TestAppointmentChangeStatus_NotFound(t *testing.T) {
	svc := newApptSvc(
		&mockAppointmentRepo{err: apperrors.ErrNotFound},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Confirm(context.Background(), 999, nil)
	assert.ErrorIs(t, err, apperrors.ErrNotFound)
}

// — canTransition (unit coverage of the state machine) —

func TestCanTransition_ValidPaths(t *testing.T) {
	cases := []struct {
		from model.AppointmentStatus
		to   model.AppointmentStatus
	}{
		{model.StatusCreated, model.StatusConfirmed},
		{model.StatusCreated, model.StatusCancelledByPatient},
		{model.StatusCreated, model.StatusCancelledByAdmin},
		{model.StatusConfirmed, model.StatusCompleted},
		{model.StatusConfirmed, model.StatusNoShow},
		{model.StatusConfirmed, model.StatusCancelledByAdmin},
	}
	for _, tc := range cases {
		assert.True(t, canTransition(tc.from, tc.to), "expected valid: %s→%s", tc.from, tc.to)
	}
}

func TestCanTransition_InvalidPaths(t *testing.T) {
	cases := []struct {
		from model.AppointmentStatus
		to   model.AppointmentStatus
	}{
		{model.StatusCompleted, model.StatusConfirmed},
		{model.StatusNoShow, model.StatusCreated},
		{model.StatusCancelledByAdmin, model.StatusCreated},
		{model.StatusCancelledByPatient, model.StatusConfirmed},
		{model.StatusConfirmed, model.StatusCreated},
		{model.StatusConfirmed, model.StatusCancelledByPatient},
	}
	for _, tc := range cases {
		assert.False(t, canTransition(tc.from, tc.to), "expected invalid: %s→%s", tc.from, tc.to)
	}
}

// — Concurrent slot protection —

// slotOnceRepo simulates a DB with EXCLUDE GIST: the first Create wins,
// all subsequent calls for the same slot return ErrSlotTaken.
type slotOnceRepo struct {
	mu      sync.Mutex
	created bool
}

func (r *slotOnceRepo) Create(_ context.Context, _ repository.CreateAppointmentInput) (*model.Appointment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.created {
		return nil, apperrors.ErrSlotTaken
	}
	r.created = true
	return &model.Appointment{
		ID: 1, DoctorID: 1, ServiceID: 1,
		Status:    model.StatusCreated,
		StartAt:   time.Now().Add(2 * time.Hour),
		EndAt:     time.Now().Add(3 * time.Hour),
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}, nil
}

func (r *slotOnceRepo) GetByID(_ context.Context, _ int64) (*repository.AppointmentDetail, error) {
	return nil, nil
}
func (r *slotOnceRepo) List(_ context.Context, _ repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return nil, nil
}
func (r *slotOnceRepo) UpdateStatus(_ context.Context, _ int64, _, _ model.AppointmentStatus, _ *int64, _ *string) error {
	return nil
}

// TestAppointmentCreate_ConcurrentSlot spawns 100 goroutines that all try to
// book the same time slot. Exactly one must succeed; all others must get
// ErrSlotTaken. Run with -race to catch data races in the service layer.
func TestAppointmentCreate_ConcurrentSlot(t *testing.T) {
	const workers = 100

	svc := NewAppointmentService(
		&slotOnceRepo{},
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
	)

	input := sampleCreateInput()

	var (
		successCount  atomic.Int64
		conflictCount atomic.Int64
		otherCount    atomic.Int64
	)

	var wg sync.WaitGroup
	wg.Add(workers)
	ready := make(chan struct{})

	for range workers {
		go func() {
			defer wg.Done()
			<-ready // all goroutines start at once
			_, err := svc.Create(context.Background(), input)
			switch {
			case err == nil:
				successCount.Add(1)
			case err == apperrors.ErrSlotTaken:
				conflictCount.Add(1)
			default:
				otherCount.Add(1)
			}
		}()
	}

	close(ready)
	wg.Wait()

	assert.Equal(t, int64(1), successCount.Load(), "exactly one goroutine must succeed")
	assert.Equal(t, int64(workers-1), conflictCount.Load(), "all other goroutines must get ErrSlotTaken")
	assert.Equal(t, int64(0), otherCount.Load(), "no unexpected errors")
}
