// Package texts holds all user-facing Lithuanian strings.
package texts

// Sheet names.
const (
	SheetSettings     = "Nustatymai"
	SheetParticipants = "Dalyviai"
	SheetEvents       = "Varžybos"
	SheetDays         = "Dienos"
	SheetReportPrefix = "Ataskaita"
)

// Settings keys.
const (
	SettingBaseURL        = "Šaltinio adresas"
	SettingSearchPath     = "Paieškos kelias"
	SettingPersonPath     = "Dalyvio rezultatų kelias"
	SettingClassPath      = "Klasės rezultatų kelias"
	SettingEventPath      = "Varžybų kelias"
	SettingProfilePath    = "Dalyvio profilio kelias"
	SettingPeriodFrom     = "Laikotarpis nuo"
	SettingPeriodTo       = "Laikotarpis iki"
	SettingDelay          = "Pauzė tarp užklausų, ms"
	SettingCacheFinalDays = "Rezultatai galutiniai po, d."
)

// Participant columns.
const (
	ColSurname = "Pavardė"
	ColName    = "Vardas"
	ColSID     = "ID"
	ColStatus  = "Būsena"
)

// Resolution statuses.
const (
	StatusResolved   = "rastas"
	StatusNotFound   = "nerastas"
	StatusAmbiguous  = "keli kandidatai"
)

// Day status display translations.
var DayStatusDisplay = map[string]string{
	"ok":          "ok",
	"dsq":         "diskv.",
	"dns":         "nestartavo",
	"dnf":         "nefinišavo",
	"mp":          "praleistas KP",
	"not entered": "nedalyvavo",
}

// Messages.
const (
	MsgPressEnter    = "Spauskite Enter, kad uždarytumėte."
	MsgCloseWorkbook = "Uždarykite reitingtuvas.xlsx ir paleiskite iš naujo."
	MsgError         = "Klaida"
)
