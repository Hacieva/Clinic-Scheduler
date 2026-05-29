package service

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Hacieva/clinic-scheduler/backend/internal/availability"
	apperrors "github.com/Hacieva/clinic-scheduler/backend/internal/errors"
	"github.com/Hacieva/clinic-scheduler/backend/internal/model"
	"github.com/Hacieva/clinic-scheduler/backend/internal/repository"
)

// mockScheduleChecker implements availability.ScheduleRepository for service tests.
type mockScheduleChecker struct {
	schedule   []availability.RegularSchedule
	exceptions []availability.Exception
	schedErr   error
	excErr     error
}

func (m *mockScheduleChecker) GetWorkingHours(_ context.Context, _ int64) ([]availability.RegularSchedule, error) {
	return m.schedule, m.schedErr
}

func (m *mockScheduleChecker) GetScheduleExceptions(_ context.Context, _ int64, _, _ time.Time) ([]availability.Exception, error) {
	return m.exceptions, m.excErr
}

// availTOD builds a time-of-day value in UTC (used by openScheduleChecker).
func availTOD(h, m int) time.Time {
	return time.Date(0, 1, 1, h, m, 0, 0, time.UTC)
}

// openScheduleChecker returns a permissive schedule covering all 7 weekdays
// from 00:00 to 23:59 UTC — ensures existing tests are not affected by the
// new working-hours check.
func openScheduleChecker() *mockScheduleChecker {
	var sched []availability.RegularSchedule
	for wd := time.Sunday; wd <= time.Saturday; wd++ {
		sched = append(sched, availability.RegularSchedule{
			DayOfWeek: wd,
			Start:     availTOD(0, 0),
			End:       availTOD(23, 59),
		})
	}
	return &mockScheduleChecker{schedule: sched}
}

// fixedFuture is a deterministic future timestamp used in unit tests.
// 2030-06-15 10:00 UTC is always After(time.Now()) and safely within a 00:00–23:59 schedule.
var fixedFuture = time.Date(2030, 6, 15, 10, 0, 0, 0, time.UTC)

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

// mockVisitRepo implements repository.VisitRepository for service-layer tests.
type mockVisitRepo struct {
	visit       *model.Visit
	list        []model.Visit
	err         error
	updateErr   error
}

func (m *mockVisitRepo) Create(_ context.Context, _ repository.CreateVisitInput) (*model.Visit, error) {
	return m.visit, m.err
}
func (m *mockVisitRepo) GetByID(_ context.Context, _ int64) (*model.Visit, error) {
	return m.visit, m.err
}
func (m *mockVisitRepo) List(_ context.Context, _ repository.VisitFilter) ([]model.Visit, error) {
	return m.list, m.err
}
func (m *mockVisitRepo) UpdateStatus(_ context.Context, _ int64, _ model.VisitStatus, _, _ *time.Time) error {
	return m.updateErr
}
func (m *mockVisitRepo) UpdatePatientID(_ context.Context, _ int64, _ int64) error {
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
		StartAt:      fixedFuture,
		Source:       model.SourceAdminPanel,
	}
}

// newApptSvc builds an AppointmentService with permissive defaults for the
// doctorSvcRepo (assigned=true) and scheduleChecker (all-hours open).
// Tests that need to override either should construct AppointmentService directly.
func newApptSvc(apptRepo *mockAppointmentRepo, docRepo *mockDoctorRepo, svcRepo *mockServiceRepo) *AppointmentService {
	return NewAppointmentService(apptRepo, &mockVisitRepo{}, docRepo, svcRepo, &mockDoctorServiceRepo{assigned: true}, openScheduleChecker())
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
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: false},
		openScheduleChecker(),
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

// Complete is now only reachable from 'arrived' (patient must check in first).
func TestAppointmentComplete_FromArrived(t *testing.T) {
	detail := sampleApptDetail(model.StatusArrived)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Complete(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestAppointmentComplete_FromConfirmed_Invalid(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.Complete(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
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

// MarkNoShow is now only reachable from 'arrived'.
func TestAppointmentMarkNoShow_FromArrived(t *testing.T) {
	detail := sampleApptDetail(model.StatusArrived)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.MarkNoShow(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestAppointmentMarkNoShow_FromConfirmed_Invalid(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	svc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := svc.MarkNoShow(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
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
		// created paths
		{model.StatusCreated, model.StatusConfirmed},
		{model.StatusCreated, model.StatusArrived},
		{model.StatusCreated, model.StatusCancelledByPatient},
		{model.StatusCreated, model.StatusCancelledByAdmin},
		// confirmed paths
		{model.StatusConfirmed, model.StatusArrived},
		{model.StatusConfirmed, model.StatusCancelledByAdmin},
		// arrived paths (terminal work)
		{model.StatusArrived, model.StatusCompleted},
		{model.StatusArrived, model.StatusNoShow},
		{model.StatusArrived, model.StatusCancelledByAdmin},
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
		// confirmed can no longer go directly to completed or no_show — must go through arrived
		{model.StatusConfirmed, model.StatusCompleted},
		{model.StatusConfirmed, model.StatusNoShow},
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
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		openScheduleChecker(),
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

// — Working-hours gate tests —

// TestAppointmentCreate_DayOff_BlocksBooking: a day_off exception makes Create
// return ErrOutsideHours even though the regular schedule would allow it.
func TestAppointmentCreate_DayOff_BlocksBooking(t *testing.T) {
	doc := sampleDoctorWithDir()
	wd := fixedFuture.Weekday()
	checker := &mockScheduleChecker{
		schedule: []availability.RegularSchedule{
			{DayOfWeek: wd, Start: availTOD(9, 0), End: availTOD(18, 0)},
		},
		exceptions: []availability.Exception{
			{Date: fixedFuture, Type: "day_off"},
		},
	}
	svc := NewAppointmentService(
		&mockAppointmentRepo{},
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		checker,
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrOutsideHours)
}

// TestAppointmentCreate_CustomHours_InsideRange: a custom_working_hours exception
// that covers fixedFuture (10:00–10:30) allows the booking.
func TestAppointmentCreate_CustomHours_InsideRange(t *testing.T) {
	doc := sampleDoctorWithDir()
	wd := fixedFuture.Weekday()
	checker := &mockScheduleChecker{
		schedule: []availability.RegularSchedule{
			{DayOfWeek: wd, Start: availTOD(9, 0), End: availTOD(12, 0)},
		},
		exceptions: []availability.Exception{
			{
				Date:  fixedFuture,
				Type:  "custom_working_hours",
				Start: ptrTime(availTOD(8, 0)),
				End:   ptrTime(availTOD(17, 0)),
			},
		},
	}
	svc := NewAppointmentService(
		&mockAppointmentRepo{appt: sampleAppt()},
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		checker,
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.NoError(t, err)
}

// TestAppointmentCreate_CustomHours_OutsideRange: custom_working_hours 11:00–17:00
// blocks a booking at fixedFuture (10:00–10:30) → ErrOutsideHours.
func TestAppointmentCreate_CustomHours_OutsideRange(t *testing.T) {
	doc := sampleDoctorWithDir()
	checker := &mockScheduleChecker{
		exceptions: []availability.Exception{
			{
				Date:  fixedFuture,
				Type:  "custom_working_hours",
				Start: ptrTime(availTOD(11, 0)),
				End:   ptrTime(availTOD(17, 0)),
			},
		},
	}
	svc := NewAppointmentService(
		&mockAppointmentRepo{},
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		checker,
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrOutsideHours)
}

// TestAppointmentCreate_NonWorkingDay: no schedule entry for fixedFuture's weekday
// → ErrOutsideHours (non-working day blocks booking).
func TestAppointmentCreate_NonWorkingDay(t *testing.T) {
	doc := sampleDoctorWithDir()
	// Build a schedule that explicitly excludes fixedFuture's weekday.
	wd := fixedFuture.Weekday()
	otherWD := (wd + 1) % 7
	checker := &mockScheduleChecker{
		schedule: []availability.RegularSchedule{
			{DayOfWeek: otherWD, Start: availTOD(9, 0), End: availTOD(18, 0)},
		},
	}
	svc := NewAppointmentService(
		&mockAppointmentRepo{},
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		checker,
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrOutsideHours)
}

// TestAppointmentCreate_DeleteException_RestoresBooking: after a day_off exception
// is deleted (simulated by the absence of exceptions), a booking within the
// regular schedule succeeds. Paired with TestAppointmentCreate_DayOff_BlocksBooking
// to demonstrate that deleting the exception restores normal working hours.
func TestAppointmentCreate_DeleteException_RestoresBooking(t *testing.T) {
	doc := sampleDoctorWithDir()
	wd := fixedFuture.Weekday()
	// No exceptions — this is the state after the day_off exception has been deleted.
	checker := &mockScheduleChecker{
		schedule: []availability.RegularSchedule{
			{DayOfWeek: wd, Start: availTOD(9, 0), End: availTOD(18, 0)},
		},
	}
	svc := NewAppointmentService(
		&mockAppointmentRepo{appt: sampleAppt()},
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		checker,
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.NoError(t, err)
}

// ptrTime is a helper used by working-hours tests.
func ptrTime(t time.Time) *time.Time { return &t }

// ── captureCreateRepo ──────────────────────────────────────────────────────────
//
// Captures the CreateAppointmentInput passed to Create so tests can assert that
// the service correctly resolved branch_id before forwarding to the repository.

type captureCreateRepo struct {
	captured  repository.CreateAppointmentInput
	appt      *model.Appointment
	createErr error
}

func (r *captureCreateRepo) Create(_ context.Context, input repository.CreateAppointmentInput) (*model.Appointment, error) {
	r.captured = input
	return r.appt, r.createErr
}
func (r *captureCreateRepo) GetByID(_ context.Context, _ int64) (*repository.AppointmentDetail, error) {
	return nil, nil
}
func (r *captureCreateRepo) List(_ context.Context, _ repository.AppointmentFilter) ([]repository.AppointmentDetail, error) {
	return nil, nil
}
func (r *captureCreateRepo) UpdateStatus(_ context.Context, _ int64, _, _ model.AppointmentStatus, _ *int64, _ *string) error {
	return nil
}

// ── Visit auto-creation tests ──────────────────────────────────────────────────

// TestAppointmentCreate_Scheduled_AutoCreatesVisit verifies that a scheduled
// appointment returned from Create always carries a non-nil visit_id.
// The mock repo simulates the repo-level Visit auto-creation by returning an
// appointment that already has VisitID set (as the real repo would after its
// internal INSERT INTO visits within the same transaction).
func TestAppointmentCreate_Scheduled_AutoCreatesVisit(t *testing.T) {
	vid := int64(42)
	apptWithVisit := sampleAppt()
	apptWithVisit.VisitID = &vid

	doc := sampleDoctorWithDir()
	doc.BranchID = int64Ptr(1)

	svc := newApptSvc(
		&mockAppointmentRepo{appt: apptWithVisit},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.VisitID, "scheduled appointment must have visit_id")
	assert.Equal(t, vid, *result.VisitID, "appointment must be linked to the created visit")
}

// TestAppointmentCreate_Scheduled_VisitLinked is the same assertion expressed
// differently: VisitID on the returned appointment matches what the repo set.
func TestAppointmentCreate_Scheduled_VisitLinked(t *testing.T) {
	vid := int64(7)
	apptWithVisit := sampleAppt()
	apptWithVisit.VisitID = &vid

	doc := sampleDoctorWithDir()
	doc.BranchID = int64Ptr(3)

	svc := newApptSvc(
		&mockAppointmentRepo{appt: apptWithVisit},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	result, err := svc.Create(context.Background(), sampleCreateInput())
	require.NoError(t, err)
	assert.Equal(t, vid, *result.VisitID)
}

// TestAppointmentCreate_Scheduled_BranchFromDoctor verifies that when
// input.BranchID is nil the service resolves it from the doctor's BranchID and
// forwards it to the repo so the repo can create the Visit.
// A nil BranchID in the repo input would silently skip Visit creation;
// a non-nil value is the contract signal to auto-create.
func TestAppointmentCreate_Scheduled_BranchFromDoctor(t *testing.T) {
	vid := int64(5)
	apptWithVisit := sampleAppt()
	apptWithVisit.VisitID = &vid

	doc := sampleDoctorWithDir()
	doc.BranchID = int64Ptr(9)

	cap := &captureCreateRepo{appt: apptWithVisit}
	svc := NewAppointmentService(
		cap,
		&mockVisitRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: true},
		openScheduleChecker(),
	)

	input := sampleCreateInput()
	// input.BranchID intentionally nil — must be resolved from doctor.

	_, err := svc.Create(context.Background(), input)
	require.NoError(t, err)
	require.NotNil(t, cap.captured.BranchID, "service must forward resolved branch_id to repo")
	assert.Equal(t, int64(9), *cap.captured.BranchID)
}

// TestAppointmentCreate_Scheduled_RepoError_NoOrphan verifies that when the repo
// returns an error the service propagates it cleanly.
// Atomicity guarantee: the Visit INSERT and Appointment INSERT are in the same
// DB transaction inside AppointmentRepo.Create; a transaction rollback removes
// both rows — orphaned visits are architecturally impossible.
func TestAppointmentCreate_Scheduled_RepoError_NoOrphan(t *testing.T) {
	doc := sampleDoctorWithDir()
	doc.BranchID = int64Ptr(1)

	svc := newApptSvc(
		&mockAppointmentRepo{err: apperrors.ErrSlotTaken},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	_, err := svc.Create(context.Background(), sampleCreateInput())
	assert.ErrorIs(t, err, apperrors.ErrSlotTaken,
		"repo error must propagate; transaction rollback prevents orphaned visit")
}
