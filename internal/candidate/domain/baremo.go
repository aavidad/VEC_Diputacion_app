package domain

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

var (
	ErrBaremoRuleSetInvalid = errors.New("baremo rule set is invalid")
	ErrBaremoMeritNoRule    = errors.New("baremo merit has no rule")
	ErrBaremoTieInvalid     = errors.New("baremo tie break is invalid")
)

type BaremoSection string

const (
	BaremoSectionExperiencia BaremoSection = "experiencia"
	BaremoSectionFormacion   BaremoSection = "formacion"
	BaremoSectionOtros       BaremoSection = "otros"
)

type BaremoUnit string

const (
	BaremoUnitMeses           BaremoUnit = "meses"
	BaremoUnitHoras           BaremoUnit = "horas"
	BaremoUnitMerito          BaremoUnit = "merito"
	BaremoUnitPuntosDeclarado BaremoUnit = "puntos_declarado"
)

type BaremoTieBreakRule string

const (
	BaremoTieMayorExperiencia BaremoTieBreakRule = "mayor_experiencia"
	BaremoTieMayorFormacion   BaremoTieBreakRule = "mayor_formacion"
	BaremoTieLetraSorteo      BaremoTieBreakRule = "letra_sorteo"
	BaremoTieCandidateID      BaremoTieBreakRule = "candidate_id"
)

type BaremoRuleSetConfig struct {
	ConvocatoriaID, Version, SorteoLetra string
	MeritRules                           []BaremoMeritRule
	SectionCaps                          []BaremoSectionCap
	TieBreakRules                        []BaremoTieBreakRule
}
type BaremoRuleSet struct {
	convocatoriaID, version, sorteoLetra string
	meritRules                           []BaremoMeritRule
	sectionCaps                          []BaremoSectionCap
	tieBreakRules                        []BaremoTieBreakRule
}
type BaremoMeritRule struct {
	MeritType     MeritType
	Section       BaremoSection
	Unit          BaremoUnit
	PointsPerUnit float64
}
type BaremoSectionCap struct {
	Section   BaremoSection
	MaxPoints float64
}
type BaremoResult struct {
	TotalPoints               float64
	SectionPoints             map[BaremoSection]float64
	RuleSetID, RuleSetVersion string
	Details                   []BaremoMeritScore
}
type BaremoMeritScore struct {
	MeritID                  string
	MeritType                MeritType
	Section                  BaremoSection
	RawPoints, AppliedPoints float64
	Capped                   bool
}
type BaremoTieCandidate struct {
	CandidateID, SorteoKey string
	Result                 BaremoResult
}
type BaremoTieBreakDecision struct {
	Rule                             BaremoTieBreakRule
	WinnerID, AValue, BValue, Reason string
}
type BaremoTieBreakResult struct {
	WinnerID  string
	IsTie     bool
	Decisions []BaremoTieBreakDecision
}

func NewBaremoRuleSet(config BaremoRuleSetConfig) (BaremoRuleSet, error) {
	ruleSet := BaremoRuleSet{
		convocatoriaID: strings.TrimSpace(config.ConvocatoriaID),
		version:        strings.TrimSpace(config.Version),
		meritRules:     append([]BaremoMeritRule(nil), config.MeritRules...),
		sectionCaps:    append([]BaremoSectionCap(nil), config.SectionCaps...),
		tieBreakRules:  append([]BaremoTieBreakRule(nil), config.TieBreakRules...),
		sorteoLetra:    strings.TrimSpace(config.SorteoLetra),
	}
	if err := ruleSet.Validate(); err != nil {
		return BaremoRuleSet{}, err
	}
	return ruleSet, nil
}

func (r BaremoRuleSet) Config() BaremoRuleSetConfig {
	return BaremoRuleSetConfig{ConvocatoriaID: r.convocatoriaID, Version: r.version, SorteoLetra: r.sorteoLetra, MeritRules: append([]BaremoMeritRule(nil), r.meritRules...), SectionCaps: append([]BaremoSectionCap(nil), r.sectionCaps...), TieBreakRules: append([]BaremoTieBreakRule(nil), r.tieBreakRules...)}
}

func (r BaremoRuleSet) Validate() error {
	if r.convocatoriaID == "" || r.version == "" || len(r.meritRules) == 0 {
		return fmt.Errorf("%w: convocatoria, version and rules are required", ErrBaremoRuleSetInvalid)
	}
	seenRules := map[MeritType]struct{}{}
	for _, rule := range r.meritRules {
		if !rule.MeritType.IsValid() || !rule.Section.IsValid() || !rule.Unit.IsValid() || !isFiniteNonNegative(rule.PointsPerUnit) {
			return fmt.Errorf("%w: invalid merit rule", ErrBaremoRuleSetInvalid)
		}
		if _, exists := seenRules[rule.MeritType]; exists {
			return fmt.Errorf("%w: duplicated merit rule %s", ErrBaremoRuleSetInvalid, rule.MeritType)
		}
		seenRules[rule.MeritType] = struct{}{}
	}
	seenCaps := map[BaremoSection]struct{}{}
	for _, cap := range r.sectionCaps {
		if !cap.Section.IsValid() || !isFiniteNonNegative(cap.MaxPoints) {
			return fmt.Errorf("%w: invalid section cap", ErrBaremoRuleSetInvalid)
		}
		if _, exists := seenCaps[cap.Section]; exists {
			return fmt.Errorf("%w: duplicated section cap %s", ErrBaremoRuleSetInvalid, cap.Section)
		}
		seenCaps[cap.Section] = struct{}{}
	}
	for _, rule := range r.tieBreakRules {
		if !rule.IsValid() {
			return fmt.Errorf("%w: invalid tie break rule", ErrBaremoRuleSetInvalid)
		}
	}
	if hasTieRule(r.tieBreakRules, BaremoTieLetraSorteo) && firstLetter(r.sorteoLetra) == "" {
		return fmt.Errorf("%w: sorteo letter is required", ErrBaremoRuleSetInvalid)
	}
	return nil
}

func CalcularAutobaremo(merits []Merit, ruleSet BaremoRuleSet) (BaremoResult, error) {
	return calcularBaremo(merits, ruleSet, func(merit Merit) bool {
		// El autobaremo es una simulacion/declaracion del aspirante. Un merito
		// rechazado o pendiente de subsanacion no puede seguir sumando, pero un
		// borrador si puede mostrarse como simulacion antes de presentar.
		return merit.Estado == MeritStateBorrador || merit.Estado == MeritStatePresentado || merit.Estado == MeritStateValidado
	})
}

// CalcularBaremoOficial solo computa decisiones administrativas validadas.
// El flujo productivo nuevo sustituira el estado mutable por decisiones
// firmadas append-only; esta funcion cierra mientras tanto el fallo del
// prototipo heredado que puntuaba rechazados y subsanaciones.
func CalcularBaremoOficial(merits []Merit, ruleSet BaremoRuleSet) (BaremoResult, error) {
	return calcularBaremo(merits, ruleSet, func(merit Merit) bool {
		return merit.Estado == MeritStateValidado
	})
}

func calcularBaremo(merits []Merit, ruleSet BaremoRuleSet, computable func(Merit) bool) (BaremoResult, error) {
	if err := ruleSet.Validate(); err != nil {
		return BaremoResult{}, err
	}
	rules, caps := ruleSet.rulesByMeritType(), ruleSet.capsBySection()
	ordered := append([]Merit(nil), merits...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].ID == ordered[j].ID {
			return ordered[i].Tipo < ordered[j].Tipo
		}
		return ordered[i].ID < ordered[j].ID
	})
	result := BaremoResult{
		RuleSetID:      ruleSet.convocatoriaID,
		RuleSetVersion: ruleSet.version,
		SectionPoints:  map[BaremoSection]float64{},
		Details:        make([]BaremoMeritScore, 0, len(ordered)),
	}
	for _, merit := range ordered {
		if err := merit.Validate(); err != nil {
			return BaremoResult{}, err
		}
		if !computable(merit) {
			continue
		}
		rule, ok := rules[merit.Tipo]
		if !ok {
			return BaremoResult{}, fmt.Errorf("%w: %s", ErrBaremoMeritNoRule, merit.Tipo)
		}
		raw := roundPoints(pointsForMerit(merit, rule))
		applied, capped := applySectionCap(raw, result.SectionPoints[rule.Section], caps[rule.Section])
		if _, hasCap := caps[rule.Section]; !hasCap {
			applied, capped = raw, false
		}
		result.SectionPoints[rule.Section] = roundPoints(result.SectionPoints[rule.Section] + applied)
		result.TotalPoints = roundPoints(result.TotalPoints + applied)
		result.Details = append(result.Details, BaremoMeritScore{
			MeritID: merit.ID, MeritType: merit.Tipo, Section: rule.Section,
			RawPoints: raw, AppliedPoints: applied, Capped: capped,
		})
	}
	return result, nil
}

func Desempate(a, b BaremoTieCandidate, ruleSet BaremoRuleSet) (BaremoTieBreakResult, error) {
	if err := ruleSet.Validate(); err != nil {
		return BaremoTieBreakResult{}, err
	}
	if strings.TrimSpace(a.CandidateID) == "" || strings.TrimSpace(b.CandidateID) == "" {
		return BaremoTieBreakResult{}, fmt.Errorf("%w: candidate ids are required", ErrBaremoTieInvalid)
	}
	decisions := make([]BaremoTieBreakDecision, 0, len(ruleSet.tieBreakRules)+1)
	for _, rule := range ruleSet.tieBreakRules {
		decision := decideTieRule(a, b, rule, ruleSet.sorteoLetra)
		decisions = append(decisions, decision)
		if decision.WinnerID != "" {
			return BaremoTieBreakResult{WinnerID: decision.WinnerID, Decisions: decisions}, nil
		}
	}
	fallback := decideCandidateID(a, b)
	decisions = append(decisions, fallback)
	return BaremoTieBreakResult{WinnerID: fallback.WinnerID, IsTie: fallback.WinnerID == "", Decisions: decisions}, nil
}

func (s BaremoSection) IsValid() bool {
	return s == BaremoSectionExperiencia || s == BaremoSectionFormacion || s == BaremoSectionOtros
}
func (u BaremoUnit) IsValid() bool {
	return u == BaremoUnitMeses || u == BaremoUnitHoras || u == BaremoUnitMerito || u == BaremoUnitPuntosDeclarado
}
func (r BaremoTieBreakRule) IsValid() bool {
	return r == BaremoTieMayorExperiencia || r == BaremoTieMayorFormacion || r == BaremoTieLetraSorteo || r == BaremoTieCandidateID
}

func (r BaremoRuleSet) rulesByMeritType() map[MeritType]BaremoMeritRule {
	rules := make(map[MeritType]BaremoMeritRule, len(r.meritRules))
	for _, rule := range r.meritRules {
		rules[rule.MeritType] = rule
	}
	return rules
}

func (r BaremoRuleSet) capsBySection() map[BaremoSection]float64 {
	caps := make(map[BaremoSection]float64, len(r.sectionCaps))
	for _, cap := range r.sectionCaps {
		caps[cap.Section] = cap.MaxPoints
	}
	return caps
}

func pointsForMerit(merit Merit, rule BaremoMeritRule) float64 {
	switch rule.Unit {
	case BaremoUnitMeses:
		return float64(merit.Datos.Meses) * rule.PointsPerUnit
	case BaremoUnitHoras:
		return float64(merit.Datos.Horas) * rule.PointsPerUnit
	case BaremoUnitPuntosDeclarado:
		return merit.Datos.PuntosFijos * rule.PointsPerUnit
	default:
		return rule.PointsPerUnit
	}
}

func applySectionCap(raw, current, cap float64) (float64, bool) {
	available := cap - current
	if available < 0 {
		available = 0
	}
	if raw > available {
		return roundPoints(available), true
	}
	return raw, false
}

func decideTieRule(a, b BaremoTieCandidate, rule BaremoTieBreakRule, sorteoLetra string) BaremoTieBreakDecision {
	switch rule {
	case BaremoTieMayorExperiencia:
		return decideGreaterSection(a, b, rule, BaremoSectionExperiencia)
	case BaremoTieMayorFormacion:
		return decideGreaterSection(a, b, rule, BaremoSectionFormacion)
	case BaremoTieLetraSorteo:
		return decideLotteryLetter(a, b, sorteoLetra)
	default:
		return decideCandidateID(a, b)
	}
}

func decideGreaterSection(a, b BaremoTieCandidate, rule BaremoTieBreakRule, section BaremoSection) BaremoTieBreakDecision {
	aValue, bValue := a.Result.SectionPoints[section], b.Result.SectionPoints[section]
	decision := BaremoTieBreakDecision{Rule: rule, AValue: formatPoints(aValue), BValue: formatPoints(bValue), Reason: "equal section points"}
	if aValue > bValue {
		decision.WinnerID, decision.Reason = a.CandidateID, "a has greater section points"
	} else if bValue > aValue {
		decision.WinnerID, decision.Reason = b.CandidateID, "b has greater section points"
	}
	return decision
}

func decideLotteryLetter(a, b BaremoTieCandidate, sorteoLetra string) BaremoTieBreakDecision {
	aLetter, bLetter := firstLetter(a.SorteoKey), firstLetter(b.SorteoKey)
	decision := BaremoTieBreakDecision{Rule: BaremoTieLetraSorteo, AValue: aLetter, BValue: bLetter, Reason: "equal lottery rank"}
	if aRank, bRank := lotteryRank(aLetter, sorteoLetra), lotteryRank(bLetter, sorteoLetra); aRank < bRank {
		decision.WinnerID, decision.Reason = a.CandidateID, "a is closer to lottery letter"
	} else if bRank < aRank {
		decision.WinnerID, decision.Reason = b.CandidateID, "b is closer to lottery letter"
	}
	return decision
}

func decideCandidateID(a, b BaremoTieCandidate) BaremoTieBreakDecision {
	aID, bID := strings.TrimSpace(a.CandidateID), strings.TrimSpace(b.CandidateID)
	decision := BaremoTieBreakDecision{Rule: BaremoTieCandidateID, AValue: aID, BValue: bID, Reason: "equal candidate ids"}
	if aID < bID {
		decision.WinnerID, decision.Reason = aID, "a has lower candidate id"
	} else if bID < aID {
		decision.WinnerID, decision.Reason = bID, "b has lower candidate id"
	}
	return decision
}

func hasTieRule(rules []BaremoTieBreakRule, target BaremoTieBreakRule) bool {
	for _, rule := range rules {
		if rule == target {
			return true
		}
	}
	return false
}

func firstLetter(value string) string {
	for _, r := range strings.ToUpper(strings.TrimSpace(value)) {
		if unicode.IsLetter(r) {
			return string(r)
		}
	}
	return ""
}

func lotteryRank(letter, start string) int {
	const maxInt = int(^uint(0) >> 1)
	alphabet := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	startRune, letterRune := []rune(firstLetter(start)), []rune(firstLetter(letter))
	if len(startRune) == 0 || len(letterRune) == 0 {
		return maxInt
	}
	startIndex, letterIndex := indexRune(alphabet, startRune[0]), indexRune(alphabet, letterRune[0])
	if startIndex < 0 || letterIndex < 0 {
		return maxInt
	}
	return (letterIndex - startIndex + len(alphabet)) % len(alphabet)
}

func indexRune(values []rune, target rune) int {
	for i, value := range values {
		if value == target {
			return i
		}
	}
	return -1
}

func isFiniteNonNegative(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value >= 0
}
func roundPoints(value float64) float64 { return math.Round(value*10000) / 10000 }
func formatPoints(value float64) string { return fmt.Sprintf("%.4f", roundPoints(value)) }
