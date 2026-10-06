package application

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"regexp"
	"strings"
	"time"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

var ErrEmisionCheckpointPeriodico = errors.New("checkpoint_periodico_emision_fallida")

var ErrCheckpointPeriodicoInvalido = errors.New("checkpoint_periodico_invalido")
var ErrCheckpointPeriodicoNoDisponible = errors.New("checkpoint_periodico_no_disponible")

var referenciaCheckpointPeriodico = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._:/#-]{2,159}$`)

type falloCheckpointPeriodico struct{ causa error }

func (falloCheckpointPeriodico) Error() string   { return ErrCheckpointPeriodicoNoDisponible.Error() }
func (e falloCheckpointPeriodico) Unwrap() error { return e.causa }
func (e falloCheckpointPeriodico) Is(err error) bool {
	return err == ErrCheckpointPeriodicoNoDisponible || errors.Is(e.causa, err)
}

type ResultadoCheckpointPeriodico struct {
	Estado  string
	Captura ports.CapturaCheckpointPeriodico
	Recibo  domain.ReciboCheckpointDesarrollo
	Acuse   ports.AcuseCheckpointPeriodico
}

func (ResultadoCheckpointPeriodico) String() string { return "[RESULTADO-CHECKPOINT-PERIODICO]" }
func (ResultadoCheckpointPeriodico) GoString() string {
	return "application.ResultadoCheckpointPeriodico{[OPACO]}"
}

// CapturarCheckpointPeriodico no calcula ventanas ni coberturas. La fuente
// debe confirmar COMMIT antes de devolver la captura auditada. Sólo entonces
// composición puede crear KMS/TSA desde la política capturada y cotejar su pin.
func CapturarCheckpointPeriodico(ctx context.Context, fuente ports.FuenteCheckpointPeriodico, maxRegistros uint64) (ports.CapturaCheckpointPeriodico, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaPeriodicaNula(fuente) || maxRegistros == 0 {
		return ports.CapturaCheckpointPeriodico{}, ErrCheckpointPeriodicoInvalido
	}
	c, err := fuente.CapturarCheckpointPendiente(ctx)
	if err != nil {
		return ports.CapturaCheckpointPeriodico{}, falloCheckpointPeriodico{err}
	}
	if ctx.Err() != nil || validarCapturaCheckpointPeriodico(c, maxRegistros) != nil {
		return ports.CapturaCheckpointPeriodico{}, ErrCheckpointPeriodicoInvalido
	}
	return c, nil
}

// SellarConfirmarCheckpointPeriodico sólo consume una captura obtenida por la
// composición confiable con la misma fuente. No admite capturas del cliente.
// No recaptura ni reintenta efectos: ante un fallo la fuente conserva el
// pendiente original. Confirmado exige el acuse de COMMIT del recibo exacto.
func SellarConfirmarCheckpointPeriodico(ctx context.Context, fuente ports.FuenteCheckpointPeriodico, c ports.CapturaCheckpointPeriodico,
	f ports.FirmadorCheckpointDesarrollo, t ports.SelladorCheckpointDesarrollo, maxRegistros uint64,
) (ResultadoCheckpointPeriodico, error) {
	if ctx == nil || ctx.Err() != nil || dependenciaPeriodicaNula(fuente) || validarCapturaCheckpointPeriodico(c, maxRegistros) != nil {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{errors.Join(ErrEmisionCheckpointPeriodico, ErrCheckpointPeriodicoInvalido)}
	}
	if c.Estado == "no_vencido" {
		return ResultadoCheckpointPeriodico{Estado: c.Estado, Acuse: c.Acuse}, nil
	}
	if dependenciaPeriodicaNula(f) || dependenciaPeriodicaNula(t) {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{errors.Join(ErrEmisionCheckpointPeriodico, ErrCheckpointPeriodicoInvalido)}
	}
	if f.PinCheckpoint() != c.PinSPKISHA256 {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{errors.Join(ErrEmisionCheckpointPeriodico, ErrCheckpointPeriodicoInvalido)}
	}
	r, err := EmitirCheckpointDesarrollo(ctx, c.Checkpoint, f, t, maxRegistros)
	if err != nil {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{errors.Join(ErrEmisionCheckpointPeriodico, err)}
	}
	raw, err := json.Marshal(r)
	if err != nil || ctx.Err() != nil {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{errors.Join(ErrEmisionCheckpointPeriodico, ErrCheckpointPeriodicoInvalido)}
	}
	a, err := fuente.ConfirmarCheckpoint(ctx, c.CapturaRef, r)
	if err != nil {
		return ResultadoCheckpointPeriodico{}, falloCheckpointPeriodico{err}
	}
	if validarAcuseCheckpointPeriodico(a) != nil || a.CapturaRef != c.CapturaRef ||
		a.ReciboHuellaSHA256 != domain.HuellaCheckpoint(raw) || a.Secuencia <= c.Acuse.Secuencia {
		return ResultadoCheckpointPeriodico{}, ErrCheckpointPeriodicoInvalido
	}
	return ResultadoCheckpointPeriodico{Estado: "confirmado", Captura: c, Recibo: r, Acuse: a}, nil
}

func validarCapturaCheckpointPeriodico(c ports.CapturaCheckpointPeriodico, maxRegistros uint64) error {
	if maxRegistros == 0 || validarAcuseCheckpointPeriodico(c.Acuse) != nil ||
		c.Acuse.CapturaRef != "" || c.Acuse.ReciboHuellaSHA256 != "" {
		return ErrCheckpointPeriodicoInvalido
	}
	switch c.Estado {
	case "no_vencido":
		if c.CapturaRef != "" || c.ConfiguracionVersion != 0 || c.ConfiguracionSHA256 != "" || c.PinSPKISHA256 != "" || c.Checkpoint != (domain.CheckpointDesarrollo{}) {
			return ErrCheckpointPeriodicoInvalido
		}
	case "pendiente":
		if !referenciaCheckpointPeriodico.MatchString(c.CapturaRef) || c.ConfiguracionVersion == 0 ||
			!huellaPeriodicaValida(c.PinSPKISHA256) ||
			!huellaPeriodicaValida(c.ConfiguracionSHA256) || c.Checkpoint.Cobertura.Registros == 0 ||
			c.Checkpoint.Cobertura.Registros > maxRegistros || !coberturaCapturaPeriodicaCoherente(c) {
			return ErrCheckpointPeriodicoInvalido
		}
		if _, err := c.Checkpoint.Canonico(); err != nil {
			return ErrCheckpointPeriodicoInvalido
		}
	default:
		return ErrCheckpointPeriodicoInvalido
	}
	return nil
}

// coberturaCapturaPeriodicaCoherente: antes de AD207 el checkpoint terminaba
// en el asiento de la propia captura. Desde AD207 cubre la cabeza sellada al
// capturar, una posición por debajo del número de ese asiento, que se sella
// después y entra en el checkpoint siguiente. Su eslabón lo coteja después el
// verificador de la cadena, no esta comprobación.
func coberturaCapturaPeriodicaCoherente(c ports.CapturaCheckpointPeriodico) bool {
	cobertura := c.Checkpoint.Cobertura
	if cobertura.UltimaSecuencia == c.Acuse.Secuencia {
		return cobertura.CabezaSHA256 == c.Acuse.HuellaSHA256
	}
	return cobertura.UltimaSecuencia < c.Acuse.Secuencia && cobertura.CabezaSHA256 != c.Acuse.HuellaSHA256
}

func validarAcuseCheckpointPeriodico(a ports.AcuseCheckpointPeriodico) error {
	if !referenciaCheckpointPeriodico.MatchString(a.AuditoriaRef) || !referenciaCheckpointPeriodico.MatchString(a.CorrelacionRef) ||
		a.Secuencia == 0 || a.Secuencia > 9007199254740991 || !huellaPeriodicaValida(a.HuellaSHA256) ||
		a.RegistradaEn.IsZero() || a.RegistradaEn.Location() != time.UTC || a.RegistradaEn.Year() < 1 ||
		a.RegistradaEn.Year() > 9999 || a.RegistradaEn.Nanosecond()%1000 != 0 {
		return ErrCheckpointPeriodicoInvalido
	}
	return nil
}

func huellaPeriodicaValida(h string) bool {
	return domain.SHA256CheckpointValido(h) && h != strings.Repeat("0", 64)
}

func dependenciaPeriodicaNula(v any) bool {
	if v == nil {
		return true
	}
	r := reflect.ValueOf(v)
	switch r.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return r.IsNil()
	}
	return false
}
