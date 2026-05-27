package importer

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// — NormalizeServiceName —

func TestNormalizeServiceName_StripLeadingCode(t *testing.T) {
	assert.Equal(t, "абляция аппаратом сургитрон без биопсии",
		NormalizeServiceName("ГИН009 Абляция аппаратом «Сургитрон» без биопсии"))
}

func TestNormalizeServiceName_ExpandAbbreviations(t *testing.T) {
	assert.Equal(t, "первичная консультация", NormalizeServiceName("перв. консультация"))
	assert.Equal(t, "повторная консультация", NormalizeServiceName("повт. консультация"))
	assert.Equal(t, "консультация гинеколога", NormalizeServiceName("к-ция гинеколога"))
}

func TestNormalizeServiceName_StripPunctuation(t *testing.T) {
	assert.Equal(t, "узи детский", NormalizeServiceName("УЗИ (детский)"))
	assert.Equal(t, "вмс удаление", NormalizeServiceName("ВМС удаление"))
}

func TestNormalizeServiceName_CollapseWhitespace(t *testing.T) {
	assert.Equal(t, "консультация невролога", NormalizeServiceName("  Консультация   невролога  "))
}

func TestNormalizeServiceName_MixedCase(t *testing.T) {
	result := NormalizeServiceName("ГИН009 Консультация гинеколога перв.")
	assert.Equal(t, "консультация гинеколога первичная", result)
}

func TestNormalizeServiceName_Empty(t *testing.T) {
	assert.Equal(t, "", NormalizeServiceName(""))
}

// — ParseWorkingHours —

func wh(dow, sh, sm, eh, em int) WorkingHoursRow {
	return WorkingHoursRow{
		DayOfWeek: dow,
		StartTime: time.Date(2000, 1, 1, sh, sm, 0, 0, time.UTC),
		EndTime:   time.Date(2000, 1, 1, eh, em, 0, 0, time.UTC),
	}
}

func TestParseWorkingHours_SingleInterval(t *testing.T) {
	rows, err := ParseWorkingHours("9:00-17:00", 1)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, wh(1, 9, 0, 17, 0), rows[0])
}

func TestParseWorkingHours_SplitDay(t *testing.T) {
	rows, err := ParseWorkingHours("9:00-13:00/14:00-18:00", 2)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, wh(2, 9, 0, 13, 0), rows[0])
	assert.Equal(t, wh(2, 14, 0, 18, 0), rows[1])
}

func TestParseWorkingHours_Vykh_LowerCase(t *testing.T) {
	rows, err := ParseWorkingHours("вых", 3)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseWorkingHours_Vykh_UpperCase(t *testing.T) {
	rows, err := ParseWorkingHours("ВЫХ", 3)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseWorkingHours_Empty(t *testing.T) {
	rows, err := ParseWorkingHours("", 4)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestParseWorkingHours_InvalidFormat(t *testing.T) {
	_, err := ParseWorkingHours("9-17", 1)
	assert.Error(t, err)
}

func TestParseWorkingHours_StartAfterEnd(t *testing.T) {
	_, err := ParseWorkingHours("17:00-9:00", 1)
	assert.Error(t, err)
}

func TestParseWorkingHours_EarlyMorning(t *testing.T) {
	rows, err := ParseWorkingHours("17:30-19:30", 5)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, wh(5, 17, 30, 19, 30), rows[0])
}

// — MapBookingType —

func TestMapBookingType_Mixed(t *testing.T) {
	kind, mode, unresolved := MapBookingType("запись_и_живая")
	assert.Equal(t, DoctorKindStaff, kind)
	assert.Equal(t, BookingModeMixed, mode)
	assert.False(t, unresolved)
}

func TestMapBookingType_AppointmentOnly(t *testing.T) {
	kind, mode, unresolved := MapBookingType("по_записи")
	assert.Equal(t, DoctorKindStaff, kind)
	assert.Equal(t, BookingModeAppointmentOnly, mode)
	assert.False(t, unresolved)
}

func TestMapBookingType_QueueOnly(t *testing.T) {
	kind, mode, unresolved := MapBookingType("по_живой")
	assert.Equal(t, DoctorKindStaff, kind)
	assert.Equal(t, BookingModeQueueOnly, mode)
	assert.False(t, unresolved)
}

func TestMapBookingType_Visiting(t *testing.T) {
	kind, mode, unresolved := MapBookingType("приезжающий")
	assert.Equal(t, DoctorKindVisiting, kind)
	assert.Equal(t, BookingModeAppointmentOnly, mode)
	assert.False(t, unresolved)
}

func TestMapBookingType_Empty(t *testing.T) {
	_, _, unresolved := MapBookingType("")
	assert.True(t, unresolved)
}

func TestMapBookingType_Unknown(t *testing.T) {
	kind, mode, unresolved := MapBookingType("неизвестно")
	assert.Equal(t, DoctorKindStaff, kind)
	assert.Equal(t, BookingModeAppointmentOnly, mode)
	assert.True(t, unresolved)
}

// — MapAudience —

func TestMapAudience_Child(t *testing.T) {
	a := MapAudience("детский")
	require.NotNil(t, a)
	assert.Equal(t, AudienceChild, *a)
}

func TestMapAudience_Adult(t *testing.T) {
	a := MapAudience("взрослый")
	require.NotNil(t, a)
	assert.Equal(t, AudienceAdult, *a)
}

func TestMapAudience_Both(t *testing.T) {
	a := MapAudience("оба")
	require.NotNil(t, a)
	assert.Equal(t, AudienceBoth, *a)
}

func TestMapAudience_Empty(t *testing.T) {
	assert.Nil(t, MapAudience(""))
}

func TestMapAudience_Unknown(t *testing.T) {
	assert.Nil(t, MapAudience("неизвестно"))
}

// — MapPatientType —

func TestMapPatientType_Children(t *testing.T) {
	assert.Equal(t, AudienceChild, MapPatientType("дети"))
}

func TestMapPatientType_Adults(t *testing.T) {
	assert.Equal(t, AudienceAdult, MapPatientType("взрослые"))
}

func TestMapPatientType_Both(t *testing.T) {
	assert.Equal(t, AudienceBoth, MapPatientType("дети/взрослые"))
}

func TestMapPatientType_Empty(t *testing.T) {
	assert.Equal(t, AudienceBoth, MapPatientType(""))
}

// — ParsePriceKopecks —

func TestParsePriceKopecks_Standard(t *testing.T) {
	v, ok := ParsePriceKopecks("7000.00")
	assert.True(t, ok)
	assert.Equal(t, int64(700000), v)
}

func TestParsePriceKopecks_NoDecimal(t *testing.T) {
	v, ok := ParsePriceKopecks("2000")
	assert.True(t, ok)
	assert.Equal(t, int64(200000), v)
}

func TestParsePriceKopecks_Empty(t *testing.T) {
	_, ok := ParsePriceKopecks("")
	assert.False(t, ok)
}

func TestParsePriceKopecks_WithSpaces(t *testing.T) {
	v, ok := ParsePriceKopecks("2 500.00")
	assert.True(t, ok)
	assert.Equal(t, int64(250000), v)
}

// — ParseDuration —

func TestParseDuration_Integer(t *testing.T) {
	n, ok := ParseDuration("30")
	assert.True(t, ok)
	assert.Equal(t, 30, n)
}

func TestParseDuration_WithPrefix(t *testing.T) {
	n, ok := ParseDuration("от 15")
	assert.True(t, ok)
	assert.Equal(t, 15, n)
}

func TestParseDuration_Empty(t *testing.T) {
	_, ok := ParseDuration("")
	assert.False(t, ok)
}

// — SplitName —

func TestSplitName_ThreeParts(t *testing.T) {
	l, f, m := SplitName("Абдулазимова Хава Зияудиновна")
	assert.Equal(t, "Абдулазимова", l)
	assert.Equal(t, "Хава", f)
	assert.Equal(t, "Зияудиновна", m)
}

func TestSplitName_TwoParts(t *testing.T) {
	l, f, m := SplitName("Гайтуркаев Ямлихан")
	assert.Equal(t, "Гайтуркаев", l)
	assert.Equal(t, "Ямлихан", f)
	assert.Equal(t, "", m)
}

func TestSplitName_Empty(t *testing.T) {
	l, f, m := SplitName("")
	assert.Equal(t, "", l)
	assert.Equal(t, "", f)
	assert.Equal(t, "", m)
}

// — ParseDirections —

func TestParseDirections_Single(t *testing.T) {
	dirs := ParseDirections("Гинеколог")
	assert.Equal(t, []string{"Гинеколог"}, dirs)
}

func TestParseDirections_Multiple(t *testing.T) {
	dirs := ParseDirections("Педиатр, пульмонолог, инфекционист")
	assert.Equal(t, []string{"Педиатр", "Пульмонолог", "Инфекционист"}, dirs)
}

func TestParseDirections_StripAudienceQualifier(t *testing.T) {
	dirs := ParseDirections("Травматолог-ортопед (детский)")
	assert.Equal(t, []string{"Травматолог-ортопед"}, dirs)
}

func TestParseDirections_Empty(t *testing.T) {
	assert.Nil(t, ParseDirections(""))
}

func TestParseDirections_Deduplication(t *testing.T) {
	dirs := ParseDirections("Гинеколог, гинеколог")
	assert.Len(t, dirs, 1)
}
