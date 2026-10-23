// Package texts holds all user-facing Lithuanian strings.
package texts

// Sheet names.
const (
	SheetSettings     = "Nustatymai"
	SheetParticipants = "Dalyviai"
	SheetEvents       = "Varžybos"
	SheetDays         = "Dienos"
	SheetResults      = "Rezultatai"
	SheetResultsLV    = "Rezultatai-LV"
	SheetManualEntry  = "Rankinis įvedimas"
	SheetReportPrefix = "Ataskaita"
)

// Settings keys.
const (
	SettingBaseURL        = "Šaltinio adresas"
	SettingSearchPath     = "Paieškos kelias"
	SettingPersonPath     = "Dalyvio rezultatų kelias"
	SettingClassPath      = "Grupės rezultatų kelias"
	SettingEventPath      = "Varžybų kelias"
	SettingProfilePath    = "Dalyvio profilio kelias"
	SettingPeriodFrom     = "Laikotarpis nuo"
	SettingPeriodTo       = "Laikotarpis iki"
	SettingDelay          = "Pauzė tarp užklausų, ms"
	SettingCacheFinalDays = "Rezultatai galutiniai po, d."
	SettingOpenClassCoef  = "Atviros klasės koeficientas"
	SettingTopN           = "Geriausių dienų skaičius"
	SettingLVBaseURL      = "LV šaltinio adresas"
	SettingLVSeriesID     = "LV serijos ID"
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

// Results sheet columns (shared by Rezultatai and Rankinis įvedimas).
const (
	ColResultSID        = "Dalyvio ID"
	ColResultEventID    = "Varžybų ID"
	ColResultDay        = "Diena"
	ColResultClass      = "Grupė"
	ColResultTime       = "Laikas"
	ColResultLeaderTime = "Lyderio laikas"
	ColResultStatus     = "Būsena"
	ColResultPlace      = "Vieta"
)

// ResultColumns is the ordered header for Rezultatai / Rankinis įvedimas.
var ResultColumns = []string{
	ColResultSID, ColResultEventID, ColResultDay, ColResultClass,
	ColResultTime, ColResultLeaderTime, ColResultStatus, ColResultPlace,
}

// Report headers.
const (
	ColReportPlace      = "Vieta"
	ColReportName       = "Pavardė Vardas"
	ColReportStarts     = "Startų skaičius"
	ColReportSeasonFmt  = "%d geriausių suma"
	ColReportEventName  = "Varžybų pavadinimas"
	ColReportDate       = "Data"
	ColReportCoef       = "Koeficientas"
	ColReportSeqNum     = "Eilės nr."
	ReportTitleFmt      = "REITINGŲ LENTELĖ %d M."
)

// Messages.
const (
	MsgPressEnter      = "Spauskite Enter, kad uždarytumėte."
	MsgCloseWorkbook   = "Uždarykite reitingtuvas.xlsx ir paleiskite iš naujo."
	MsgError           = "Klaida"
	MsgPromptDownload  = "Ar norite parsisiųsti duomenis iš šaltinių? (t/n): "
	MsgPromptOverwrite = "Perrašyti paskutinę ataskaitą ar sukurti naują? (p/n): "
	MsgNoResults       = "Nėra rezultatų. Pirmiausia parsisiųskite duomenis."
)
