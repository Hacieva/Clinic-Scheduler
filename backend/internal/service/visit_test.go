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

// ── helpers ────────────────────────────────────────────────────────────────────

func sampleVisit() *model.Visit {
	return &model.Visit{
		ID:        1,
		PatientID: 99,
		BranchID:  1,
		VisitType: model.VisitTypeWalkIn,
		Status:    model.VisitStatusInProgress,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func sampleWalkInInput() RegisterWalkInInput {
	return RegisterWalkInInput{
		PatientName:  "Walk In Patient",
		PatientPhone: "+70000000001",
		DoctorID:     1,
		ServiceID:    1,
		BranchID:     1,
		Source:       model.SourceAdminPanel,
	}
}

func newVisitSvc(
	visitRepo *mockVisitRepo,
	apptRepo *mockAppointmentRepo,
	patientRepo *mockPatientRepo,
	docRepo *mockDoctorRepo,
	svcRepo *mockServiceRepo,
) *VisitService {
	return NewVisitService(
		visitRepo,
		apptRepo,
		patientRepo,
		docRepo,
		svcRepo,
		&mockDoctorServiceRepo{assigned: true},
	)
}

// ── RegisterWalkIn tests ───────────────────────────────────────────────────────

func TestRegisterWalkIn_Success(t *testing.T) {
	appt := sampleAppt()
	appt.AppointmentType = model.AppointmentTypeWalkIn
	appt.Status = model.StatusArrived

	svc := newVisitSvc(
		&mockVisitRepo{visit: sampleVisit()},
		&mockAppointmentRepo{appt: appt},
		&mockPatientRepo{byPhoneErr: apperrors.ErrNotFound, patient: &model.Patient{ID: 99, FullName: "Test", Phone: "+70000000001", Source: "admin_panel"}}, // forces patient creation
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
	)

	visit, appointment, err := svc.RegisterWalkIn(context.Background(), sampleWalkInInput())
	require.NoError(t, err)
	require.NotNil(t, visit)
	require.NotNil(t, appointment)

	assert.Equal(t, model.VisitStatusInProgress, visit.Status)
	assert.Equal(t, model.VisitTypeWalkIn, visit.VisitType)
	assert.Equal(t, model.StatusArrived, appointment.Status)
	assert.Equal(t, model.AppointmentTypeWalkIn, appointment.AppointmentType)
}

func TestRegisterWalkIn_ExistingPatient(t *testing.T) {
	existingPatient := &model.Patient{ID: 7, FullName: "Existing", Phone: "+70000000001"}
	appt := sampleAppt()
	appt.AppointmentType = model.AppointmentTypeWalkIn
	appt.Status = model.StatusArrived

	svc := newVisitSvc(
		&mockVisitRepo{visit: sampleVisit()},
		&mockAppointmentRepo{appt: appt},
		&mockPatientRepo{patient: existingPatient}, // existing patient found by phone
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
	)

	visit, appointment, err := svc.RegisterWalkIn(context.Background(), sampleWalkInInput())
	require.NoError(t, err)
	require.NotNil(t, visit)
	require.NotNil(t, appointment)
}

func TestRegisterWalkIn_DoctorInactive(t *testing.T) {
	inactiveDoc := sampleDoctorWithDir()
	inactiveDoc.IsActive = false

	svc := newVisitSvc(
		&mockVisitRepo{},
		&mockAppointmentRepo{},
		&mockPatientRepo{},
		&mockDoctorRepo{doctor: inactiveDoc},
		&mockServiceRepo{svc: activeSvc()},
	)

	_, _, err := svc.RegisterWalkIn(context.Background(), sampleWalkInInput())
	assert.ErrorIs(t, err, apperrors.ErrDoctorInactive)
}

func TestRegisterWalkIn_AppointmentOnlyMode(t *testing.T) {
	doc := sampleDoctorWithDir()
	doc.BookingMode = "appointment_only"

	svc := newVisitSvc(
		&mockVisitRepo{},
		&mockAppointmentRepo{},
		&mockPatientRepo{},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	_, _, err := svc.RegisterWalkIn(context.Background(), sampleWalkInInput())
	assert.ErrorIs(t, err, apperrors.ErrInvalidBookingMode)
}

func TestRegisterWalkIn_MissingPatientName(t *testing.T) {
	input := sampleWalkInInput()
	input.PatientName = ""

	svc := newVisitSvc(
		&mockVisitRepo{},
		&mockAppointmentRepo{},
		&mockPatientRepo{},
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
	)

	_, _, err := svc.RegisterWalkIn(context.Background(), input)
	assert.ErrorIs(t, err, apperrors.ErrInvalidInput)
}

func TestRegisterWalkIn_ServiceNotAssigned(t *testing.T) {
	svc := NewVisitService(
		&mockVisitRepo{},
		&mockAppointmentRepo{},
		&mockPatientRepo{},
		&mockDoctorRepo{doctor: sampleDoctorWithDir()},
		&mockServiceRepo{svc: activeSvc()},
		&mockDoctorServiceRepo{assigned: false}, // not assigned
	)

	_, _, err := svc.RegisterWalkIn(context.Background(), sampleWalkInInput())
	assert.ErrorIs(t, err, apperrors.ErrDirectionMismatch)
}

// ── Arrive + syncVisitStatus tests ────────────────────────────────────────────

func TestArrive_TransitionsToArrived(t *testing.T) {
	detail := sampleApptDetail(model.StatusConfirmed)
	detail.VisitID = func() *int64 { v := int64(1); return &v }()

	apptSvc := NewAppointmentService(
		&mockAppointmentRepo{detail: detail},
		&mockVisitRepo{},
		&mockDoctorRepo{},
		&mockServiceRepo{},
		&mockDoctorServiceRepo{assigned: true},
		openScheduleChecker(),
	)

	err := apptSvc.Arrive(context.Background(), 1, nil)
	require.NoError(t, err)
}

func TestArrive_FromCompleted_Invalid(t *testing.T) {
	detail := sampleApptDetail(model.StatusCompleted)
	detail.VisitID = func() *int64 { v := int64(1); return &v }()

	apptSvc := newApptSvc(
		&mockAppointmentRepo{detail: detail},
		&mockDoctorRepo{},
		&mockServiceRepo{},
	)

	err := apptSvc.Arrive(context.Background(), 1, nil)
	assert.ErrorIs(t, err, apperrors.ErrInvalidStatusTransition)
}

// ── booking_mode enforcement tests ────────────────────────────────────────────

func TestCreate_QueueOnlyRejectsScheduled(t *testing.T) {
	doc := sampleDoctorWithDir()
	doc.BookingMode = "queue_only"

	svc := newApptSvc(
		&mockAppointmentRepo{appt: sampleAppt()},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	input := sampleCreateInput()
	input.AppointmentType = model.AppointmentTypeScheduled
	_, err := svc.Create(context.Background(), input)
	assert.ErrorIs(t, err, apperrors.ErrInvalidBookingMode)
}

func TestCreate_AppointmentOnlyRejectsWalkIn(t *testing.T) {
	doc := sampleDoctorWithDir()
	doc.BookingMode = "appointment_only"

	svc := newApptSvc(
		&mockAppointmentRepo{appt: sampleAppt()},
		&mockDoctorRepo{doctor: doc},
		&mockServiceRepo{svc: activeSvc()},
	)

	input := sampleCreateInput()
	input.AppointmentType = model.AppointmentTypeWalkIn
	_, err := svc.Create(context.Background(), input)
	assert.ErrorIs(t, err, apperrors.ErrInvalidBookingMode)
}

func TestCreate_MixedAllowsBoth(t *testing.T) {
	doc := sampleDoctorWithDir()
	doc.BookingMode = "mixed"

	apptRepo := &mockAppointmentRepo{appt: sampleAppt()}
	svc := newApptSvc(apptRepo, &mockDoctorRepo{doctor: doc}, &mockServiceRepo{svc: activeSvc()})

	inputScheduled := sampleCreateInput()
	inputScheduled.AppointmentType = model.AppointmentTypeScheduled
	_, err := svc.Create(context.Background(), inputScheduled)
	require.NoError(t, err)
}
