package availability

import (
	"context"
	"time"
)

// ScheduleRepository provides doctor working hours and schedule exceptions.
type ScheduleRepository interface {
	GetWorkingHours(ctx context.Context, doctorID int64) ([]RegularSchedule, error)
	GetScheduleExceptions(ctx context.Context, doctorID int64, from, to time.Time) ([]Exception, error)
}

// AppointmentRepository provides existing bookings as slots.
type AppointmentRepository interface {
	GetSlotsByDoctor(ctx context.Context, doctorID int64, from, to time.Time) ([]Slot, error)
}

// ServiceRepository provides the duration of a medical service.
type ServiceRepository interface {
	GetDurationMinutes(ctx context.Context, serviceID int64) (int, error)
}

// Service assembles repository data and delegates slot calculation to Calculate.
type Service struct {
	scheduleRepo ScheduleRepository
	apptRepo     AppointmentRepository
	serviceRepo  ServiceRepository
}

func NewService(sr ScheduleRepository, ar AppointmentRepository, svcr ServiceRepository) *Service {
	return &Service{
		scheduleRepo: sr,
		apptRepo:     ar,
		serviceRepo:  svcr,
	}
}

// GetAvailability returns available slots for doctorID/serviceID across [from, to].
func (s *Service) GetAvailability(
	ctx context.Context,
	doctorID, serviceID int64,
	from, to time.Time,
) ([]DayAvailability, error) {
	durationMin, err := s.serviceRepo.GetDurationMinutes(ctx, serviceID)
	if err != nil {
		return nil, err
	}

	schedule, err := s.scheduleRepo.GetWorkingHours(ctx, doctorID)
	if err != nil {
		return nil, err
	}

	exceptions, err := s.scheduleRepo.GetScheduleExceptions(ctx, doctorID, from, to)
	if err != nil {
		return nil, err
	}

	booked, err := s.apptRepo.GetSlotsByDoctor(ctx, doctorID, from, to)
	if err != nil {
		return nil, err
	}

	var result []DayAvailability
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		day := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())

		var dayBooked []Slot
		for _, a := range booked {
			if sameDay(day, a.Start) {
				dayBooked = append(dayBooked, a)
			}
		}

		slots := Calculate(CalculatorInput{
			Date:                 day,
			ServiceDuration:      time.Duration(durationMin) * time.Minute,
			RegularSchedule:      schedule,
			Exceptions:           exceptions,
			ExistingAppointments: dayBooked,
			SlotStep:             30 * time.Minute,
		})

		if len(slots) > 0 {
			result = append(result, DayAvailability{Date: day, Slots: slots})
		}
	}

	return result, nil
}
