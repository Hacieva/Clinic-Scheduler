package availability

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockScheduleRepo struct {
	workingHours []RegularSchedule
	exceptions   []Exception
	workErr      error
	excErr       error
}

func (m *mockScheduleRepo) GetWorkingHours(_ context.Context, _ int64) ([]RegularSchedule, error) {
	return m.workingHours, m.workErr
}

func (m *mockScheduleRepo) GetScheduleExceptions(_ context.Context, _ int64, _, _ time.Time) ([]Exception, error) {
	return m.exceptions, m.excErr
}

type mockApptRepo struct {
	slots []Slot
	err   error
}

func (m *mockApptRepo) GetSlotsByDoctor(_ context.Context, _ int64, _, _ time.Time) ([]Slot, error) {
	return m.slots, m.err
}

type mockServiceRepo struct {
	duration int
	err      error
}

func (m *mockServiceRepo) GetDurationMinutes(_ context.Context, _ int64) (int, error) {
	return m.duration, m.err
}

var wedSchedule = []RegularSchedule{
	{DayOfWeek: time.Wednesday, Start: tod(10, 0), End: tod(12, 0)},
}

func newTestSvc(sched []RegularSchedule, exc []Exception, slots []Slot, dur int) *Service {
	return NewService(
		&mockScheduleRepo{workingHours: sched, exceptions: exc},
		&mockApptRepo{slots: slots},
		&mockServiceRepo{duration: dur},
	)
}

// TestGetAvailability_Basic: single day, no bookings → 3 slots, 1 DayAvailability.
func TestGetAvailability_Basic(t *testing.T) {
	svc := newTestSvc(wedSchedule, nil, nil, 60)

	result, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, refDate, result[0].Date)
	assert.Len(t, result[0].Slots, 3)
}

// TestGetAvailability_WithBooking: 10:00–11:00 booked → only 11:00–12:00 returned.
func TestGetAvailability_WithBooking(t *testing.T) {
	svc := newTestSvc(wedSchedule, nil, []Slot{{Start: dt(10, 0), End: dt(11, 0)}}, 60)

	result, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.NoError(t, err)
	require.Len(t, result, 1)
	require.Len(t, result[0].Slots, 1)
	assert.Equal(t, dt(11, 0), result[0].Slots[0].Start)
}

// TestGetAvailability_DayOff: day_off exception → no DayAvailability entry.
func TestGetAvailability_DayOff(t *testing.T) {
	exc := []Exception{{Date: refDate, Type: "day_off"}}
	svc := newTestSvc(wedSchedule, exc, nil, 60)

	result, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetAvailability_AllBooked: all slots taken → day omitted from result.
func TestGetAvailability_AllBooked(t *testing.T) {
	booked := []Slot{
		{Start: dt(10, 0), End: dt(11, 0)},
		{Start: dt(10, 30), End: dt(11, 30)},
		{Start: dt(11, 0), End: dt(12, 0)},
	}
	svc := newTestSvc(wedSchedule, nil, booked, 60)

	result, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetAvailability_MultiDay: two-day range; only Wednesday has schedule → 1 entry.
func TestGetAvailability_MultiDay(t *testing.T) {
	svc := newTestSvc(wedSchedule, nil, nil, 60)
	to := refDate.AddDate(0, 0, 1)

	result, err := svc.GetAvailability(context.Background(), 1, 1, refDate, to)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, refDate, result[0].Date)
}

// TestGetAvailability_FromAfterTo: from > to → empty result, no error.
func TestGetAvailability_FromAfterTo(t *testing.T) {
	svc := newTestSvc(wedSchedule, nil, nil, 60)

	result, err := svc.GetAvailability(context.Background(), 1, 1,
		refDate.AddDate(0, 0, 1), refDate)

	require.NoError(t, err)
	assert.Empty(t, result)
}

// TestGetAvailability_ServiceRepoError: GetDurationMinutes fails → propagate error.
func TestGetAvailability_ServiceRepoError(t *testing.T) {
	svc := NewService(
		&mockScheduleRepo{},
		&mockApptRepo{},
		&mockServiceRepo{err: errors.New("db error")},
	)

	_, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.Error(t, err)
}

// TestGetAvailability_ScheduleError: GetWorkingHours fails → propagate error.
func TestGetAvailability_ScheduleError(t *testing.T) {
	svc := NewService(
		&mockScheduleRepo{workErr: errors.New("db error")},
		&mockApptRepo{},
		&mockServiceRepo{duration: 60},
	)

	_, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.Error(t, err)
}

// TestGetAvailability_ExceptionsError: GetScheduleExceptions fails → propagate error.
func TestGetAvailability_ExceptionsError(t *testing.T) {
	svc := NewService(
		&mockScheduleRepo{excErr: errors.New("db error")},
		&mockApptRepo{},
		&mockServiceRepo{duration: 60},
	)

	_, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.Error(t, err)
}

// TestGetAvailability_BookingsError: GetSlotsByDoctor fails → propagate error.
func TestGetAvailability_BookingsError(t *testing.T) {
	svc := NewService(
		&mockScheduleRepo{workingHours: wedSchedule},
		&mockApptRepo{err: errors.New("db error")},
		&mockServiceRepo{duration: 60},
	)

	_, err := svc.GetAvailability(context.Background(), 1, 1, refDate, refDate)

	require.Error(t, err)
}
