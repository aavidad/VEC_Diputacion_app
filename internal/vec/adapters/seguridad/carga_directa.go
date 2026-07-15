package seguridad

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"vec-diputacion-granada/internal/vec/ports"
)

var (
	ErrConfiguracionCriptografiaCargaDirectaInvalida = errors.New("seguridad: configuracion criptografica de carga directa invalida")
	ErrMaterialCriptograficoCargaDirectaInvalido     = errors.New("seguridad: material criptografico de carga directa invalido")
	ErrCriptografiaCargaDirectaCerrada               = errors.New("seguridad: criptografia de carga directa cerrada")
	ErrEntropiaCargaDirectaNoDisponible              = errors.New("seguridad: entropia de carga directa no disponible")
)

const (
	longitudMinimaClaveHMACCargaDirecta = 32
	bytesAleatoriosReciboCargaDirecta   = 32 // 256 bits reales.
	bytesEvidenciaReciboCargaDirecta    = 32
	maximoIntentosAltaRecibo            = 3
	duracionMaximaReciboCargaDirecta    = 10 * time.Minute
	versionReciboCargaDirecta           = "rcd2"
)

// ConfiguracionClaveHMACCargaDirecta transporta material desde el ensamblado.
// Sus representaciones nunca muestran la clave y el constructor conserva una
// copia propia. En produccion el origen debe ser KMS, HSM o gestor de secretos.
type ConfiguracionClaveHMACCargaDirecta struct {
	Identificador string `json:"identificador"`
	Material      []byte `json:"-"`
}

func (ConfiguracionClaveHMACCargaDirecta) String() string {
	return "seguridad.ConfiguracionClaveHMACCargaDirecta{[MATERIAL-OCULTO]}"
}

func (ConfiguracionClaveHMACCargaDirecta) GoString() string {
	return "seguridad.ConfiguracionClaveHMACCargaDirecta{[MATERIAL-OCULTO]}"
}

func (c ConfiguracionClaveHMACCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ConfiguracionClaveHMACCargaDirecta) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"clave_hmac_carga_directa","material":"oculto"}`), nil
}

func (ConfiguracionClaveHMACCargaDirecta) MarshalText() ([]byte, error) {
	return []byte("[MATERIAL-HMAC-CARGA-DIRECTA-OCULTO]"), nil
}

func (c ConfiguracionClaveHMACCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

// ConfiguracionCriptografiaCargaDirecta separa cuatro finalidades que no pueden
// compartir identificador ni material: seudonimizar sujetos, indexar recibos
// autenticar el vinculo inmutable y atestar el consumo durable.
type ConfiguracionCriptografiaCargaDirecta struct {
	ClaveSeudonimizacion ConfiguracionClaveHMACCargaDirecta `json:"clave_seudonimizacion"`
	ClaveIndiceRecibo    ConfiguracionClaveHMACCargaDirecta `json:"clave_indice_recibo"`
	ClaveVinculoRecibo   ConfiguracionClaveHMACCargaDirecta `json:"clave_vinculo_recibo"`
	ClaveAtestacion      ConfiguracionClaveHMACCargaDirecta `json:"clave_atestacion"`
}

func (ConfiguracionCriptografiaCargaDirecta) String() string {
	return "seguridad.ConfiguracionCriptografiaCargaDirecta{[MATERIAL-OCULTO]}"
}

func (ConfiguracionCriptografiaCargaDirecta) GoString() string {
	return "seguridad.ConfiguracionCriptografiaCargaDirecta{[MATERIAL-OCULTO]}"
}

func (c ConfiguracionCriptografiaCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, c.String())
}

func (ConfiguracionCriptografiaCargaDirecta) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"configuracion_criptografia_carga_directa","material":"oculto"}`), nil
}

func (ConfiguracionCriptografiaCargaDirecta) MarshalText() ([]byte, error) {
	return []byte("[CONFIGURACION-CRIPTOGRAFIA-CARGA-DIRECTA-OCULTA]"), nil
}

func (c ConfiguracionCriptografiaCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(c.String())
}

type claveHMACCargaDirecta struct {
	identificador string
	material      []byte
}

// AdaptadorCriptograficoCargaDirecta no conserva recibos. Emite 256 bits
// aleatorios, persiste exclusivamente su indice HMAC y delega el uso unico en
// un repositorio durable con consumo condicional atomico.
type AdaptadorCriptograficoCargaDirecta struct {
	mu                   sync.RWMutex
	claveSeudonimizacion claveHMACCargaDirecta
	claveIndiceRecibo    claveHMACCargaDirecta
	claveVinculoRecibo   claveHMACCargaDirecta
	claveAtestacion      claveHMACCargaDirecta
	repositorio          ports.RepositorioRecibosCargaDirecta
	reloj                ports.Reloj
	cerrado              bool
}

func NuevoAdaptadorCriptograficoCargaDirecta(
	configuracion ConfiguracionCriptografiaCargaDirecta,
	repositorio ports.RepositorioRecibosCargaDirecta,
	reloj ports.Reloj,
) (*AdaptadorCriptograficoCargaDirecta, error) {
	if !configuracionCriptografiaCargaDirectaValida(configuracion) ||
		dependenciaCargaDirectaNula(repositorio) || dependenciaCargaDirectaNula(reloj) {
		return nil, ErrConfiguracionCriptografiaCargaDirectaInvalida
	}
	return &AdaptadorCriptograficoCargaDirecta{
		claveSeudonimizacion: copiarClaveHMACCargaDirecta(configuracion.ClaveSeudonimizacion),
		claveIndiceRecibo:    copiarClaveHMACCargaDirecta(configuracion.ClaveIndiceRecibo),
		claveVinculoRecibo:   copiarClaveHMACCargaDirecta(configuracion.ClaveVinculoRecibo),
		claveAtestacion:      copiarClaveHMACCargaDirecta(configuracion.ClaveAtestacion),
		repositorio:          repositorio,
		reloj:                reloj,
	}, nil
}

func (a *AdaptadorCriptograficoCargaDirecta) SeudonimizarSujetoAlmacen(
	ctx context.Context,
	solicitud ports.SolicitudSeudonimizarSujetoAlmacen,
) (string, error) {
	if err := validarContextoCargaDirecta(ctx); err != nil {
		return "", err
	}
	sujetoRef, ambitoRef, err := solicitud.RevelarParaSellado()
	if err != nil {
		return "", ports.ErrSeudonimizacionAlmacenNoDisponible
	}
	if a == nil {
		return "", ErrConfiguracionCriptografiaCargaDirectaInvalida
	}
	instantanea, err := a.obtenerInstantanea()
	if err != nil {
		return "", err
	}
	defer instantanea.borrar()
	suma := calcularHMACCargaDirecta(
		instantanea.claveSeudonimizacion,
		"seudonimo-sujeto-almacen-v1",
		sujetoRef,
		ambitoRef,
	)
	defer borrarBytesCargaDirecta(suma)
	return representarHMACCargaDirecta(instantanea.claveSeudonimizacion.identificador, suma), nil
}

func (a *AdaptadorCriptograficoCargaDirecta) EmitirReciboCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudEmitirReciboCargaDirecta,
) (ports.ReciboCargaDirecta, error) {
	if err := validarContextoCargaDirecta(ctx); err != nil {
		return ports.ReciboCargaDirecta{}, err
	}
	contexto, sesionRef, expiraEn, vinculoSolicitudSHA256, err := solicitud.RevelarParaEmision()
	if err != nil {
		return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	instantanea, err := a.obtenerInstantanea()
	if err != nil {
		return ports.ReciboCargaDirecta{}, err
	}
	defer instantanea.borrar()
	// Reloj y repositorio se invocan fuera del bloqueo de claves. La copia
	// efimera se borra al concluir la operacion.
	// Esta lectura solo evita iniciar una concesion evidentemente vencida. No
	// se persiste ni se presenta como evidencia: la fecha probatoria la elige
	// el repositorio dentro del alta durable.
	observadoEnProceso := instantanea.reloj.Ahora()
	if observadoEnProceso.IsZero() || !expiraEn.After(observadoEnProceso) ||
		observadoEnProceso.Location() != time.UTC || expiraEn.Location() != time.UTC ||
		expiraEn.Sub(observadoEnProceso) > duracionMaximaReciboCargaDirecta {
		return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}

	for intento := 0; intento < maximoIntentosAltaRecibo; intento++ {
		if err := validarContextoCargaDirecta(ctx); err != nil {
			return ports.ReciboCargaDirecta{}, err
		}
		materialRecibo, registro, err := crearMaterialReciboYRegistro(
			instantanea.claveIndiceRecibo, instantanea.claveVinculoRecibo,
			contexto, sesionRef, vinculoSolicitudSHA256, expiraEn,
		)
		if err != nil {
			return ports.ReciboCargaDirecta{}, err
		}
		// La escritura durable del recibo es el efecto. Se revalida la
		// capacidad con el reloj inyectado inmediatamente antes de ejecutarlo.
		if err := contexto.ValidarParaEn(
			ports.AccionAlmacenPrepararCargaDirecta, instantanea.reloj.Ahora(),
		); err != nil {
			return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
		}
		resultadoAlta, err := instantanea.repositorio.RegistrarReciboCargaDirecta(ctx, registro)
		if err == nil {
			if resultadoAlta.ValidarContra(registro) != nil {
				return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
			}
			recibo, err := completarReciboTrasAltaDurable(
				instantanea.claveVinculoRecibo, materialRecibo, registro, resultadoAlta.RegistradoEn,
			)
			if err != nil {
				return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
			}
			return recibo, nil
		}
		if erroresContextoCargaDirecta(ctx, err) != nil {
			return ports.ReciboCargaDirecta{}, erroresContextoCargaDirecta(ctx, err)
		}
		if !errors.Is(err, ports.ErrRegistroReciboCargaDirectaConflicto) {
			return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
		}
	}
	return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
}

func (a *AdaptadorCriptograficoCargaDirecta) ConsumirReciboCargaDirecta(
	ctx context.Context,
	solicitud ports.SolicitudConsumirReciboCargaDirecta,
) (ports.ComprobanteConsumoReciboCargaDirecta, error) {
	if err := validarContextoCargaDirecta(ctx); err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	if solicitud.Validar() != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	valorRecibo, err := solicitud.Recibo.RevelarParaEntregaOConsumo()
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	instantanea, err := a.obtenerInstantanea()
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	defer instantanea.borrar()
	partes, err := analizarReciboCargaDirecta(valorRecibo, instantanea.claveVinculoRecibo.identificador)
	if err != nil {
		partes.borrar()
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	defer partes.borrar()

	vinculoCalculado := calcularVinculoReciboCargaDirecta(
		instantanea.claveVinculoRecibo,
		solicitud.Contexto,
		solicitud.SesionRef,
		partes.vinculoSolicitudSHA256,
		partes.expiraEn,
		partes.aleatorio,
	)
	defer borrarBytesCargaDirecta(vinculoCalculado)
	if !hmac.Equal(vinculoCalculado, partes.vinculo) {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	vinculoHMAC := representarHMACCargaDirecta(instantanea.claveVinculoRecibo.identificador, vinculoCalculado)
	indice := calcularHMACCargaDirecta(
		instantanea.claveIndiceRecibo, "indice-recibo-carga-directa-v1", partes.materialIndice,
	)
	defer borrarBytesCargaDirecta(indice)
	grupo := calcularGrupoReciboCargaDirecta(
		instantanea.claveIndiceRecibo, solicitud.Contexto, solicitud.SesionRef,
	)
	defer borrarBytesCargaDirecta(grupo)
	indiceHMAC := representarHMACCargaDirecta(instantanea.claveIndiceRecibo.identificador, indice)
	grupoHMAC := representarHMACCargaDirecta(instantanea.claveIndiceRecibo.identificador, grupo)
	atestacionAlta := calcularAtestacionAltaReciboCargaDirecta(
		instantanea.claveVinculoRecibo, partes.materialIndice, indiceHMAC, grupoHMAC, partes.registradoEn,
	)
	defer borrarBytesCargaDirecta(atestacionAlta)
	if !hmac.Equal(atestacionAlta, partes.atestacionAlta) {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	evidenciaRef, err := nuevaReferenciaOpacaCargaDirecta("recibo-consumo-v1")
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	intencionRef, err := nuevaReferenciaOpacaCargaDirecta("confirmacion-intencion-v1")
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, err
	}
	huellaIntencion := calcularHuellaIntencionCargaDirecta(
		instantanea.claveAtestacion, solicitud.Contexto, solicitud.SesionRef,
		vinculoHMAC, indiceHMAC, grupoHMAC,
		intencionRef, evidenciaRef, partes.registradoEn, solicitud.ValidaHasta,
	)
	defer borrarBytesCargaDirecta(huellaIntencion)
	orden := ports.OrdenConsumoReciboCargaDirecta{
		IndiceHMAC:               indiceHMAC,
		GrupoHMAC:                grupoHMAC,
		VinculoHMAC:              vinculoHMAC,
		EvidenciaConsumoRef:      evidenciaRef,
		IntencionConfirmacionRef: intencionRef,
		HuellaIntencionHMAC:      representarHMACCargaDirecta(instantanea.claveAtestacion.identificador, huellaIntencion),
		RegistradoEn:             partes.registradoEn,
		ValidaHasta:              solicitud.ValidaHasta,
	}
	if orden.Validar() != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	if err := solicitud.Contexto.ValidarParaEn(
		ports.AccionAlmacenConfirmarCargaDirecta, instantanea.reloj.Ahora(),
	); err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	resultado, err := instantanea.repositorio.ConsumirReciboCargaDirecta(ctx, orden)
	if err != nil {
		if errContexto := erroresContextoCargaDirecta(ctx, err); errContexto != nil {
			return ports.ComprobanteConsumoReciboCargaDirecta{}, errContexto
		}
		if errors.Is(err, ports.ErrConsumoReciboCargaDirectaDenegado) {
			return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
		}
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	if resultado.ValidarContra(orden) != nil || !resultado.ExpiraEn.Equal(partes.expiraEn) {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	atestacion := calcularAtestacionConsumoCargaDirecta(
		instantanea.claveAtestacion, solicitud.Contexto, solicitud.SesionRef, resultado, solicitud.ValidaHasta,
	)
	defer borrarBytesCargaDirecta(atestacion)
	comprobante, err := ports.NuevoComprobanteConsumoReciboCargaDirecta(
		solicitud, resultado,
		representarHMACCargaDirecta(instantanea.claveAtestacion.identificador, atestacion),
	)
	if err != nil {
		return ports.ComprobanteConsumoReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	return comprobante, nil
}

// VerificarAtestacionConsumoReciboCargaDirecta es la segunda fase obligatoria
// antes de que el nucleo construya una solicitud para el conector. Verifica
// con una cuarta clave exclusiva el contexto completo de confirmacion,
// incluidos AutorizacionRef, Accion, sesion y la fecha durable.
func (a *AdaptadorCriptograficoCargaDirecta) VerificarAtestacionConsumoReciboCargaDirecta(
	ctx context.Context,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
	comprobante ports.ComprobanteConsumoReciboCargaDirecta,
) error {
	if err := validarContextoCargaDirecta(ctx); err != nil {
		return err
	}
	if !referenciaCanonicaCargaDirectaValida(sesionRef, 512) {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	indice, grupo, vinculo, evidencia, intencion, huellaIntencion,
		registradoEn, consumidoEn, expiraEn, validaHasta, atestacionHMAC, err := comprobante.RevelarParaVerificacion()
	if err != nil {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	instantanea, err := a.obtenerInstantanea()
	if err != nil {
		return err
	}
	defer instantanea.borrar()
	if contexto.ValidarParaEn(
		ports.AccionAlmacenConfirmarCargaDirecta, instantanea.reloj.Ahora(),
	) != nil {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	resultado := ports.ResultadoConsumoReciboCargaDirecta{
		IndiceHMAC: indice, GrupoHMAC: grupo, VinculoHMAC: vinculo,
		EvidenciaConsumoRef: evidencia, IntencionConfirmacionRef: intencion,
		HuellaIntencionHMAC: huellaIntencion, RegistradoEn: registradoEn,
		ConsumidoEn: consumidoEn, ExpiraEn: expiraEn,
	}
	if resultado.Validar() != nil || !consumidoEn.Before(validaHasta) {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	calculada := calcularAtestacionConsumoCargaDirecta(
		instantanea.claveAtestacion, contexto, sesionRef, resultado, validaHasta,
	)
	defer borrarBytesCargaDirecta(calculada)
	if !hmacRepresentadoCargaDirectaValido(
		atestacionHMAC, instantanea.claveAtestacion.identificador, calculada,
	) {
		return ports.ErrAtestacionReciboCargaDirectaNoValida
	}
	return nil
}

func crearMaterialReciboYRegistro(
	claveIndiceRecibo, claveVinculoRecibo claveHMACCargaDirecta,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef, vinculoSolicitudSHA256 string,
	expiraEn time.Time,
) (string, ports.RegistroReciboCargaDirecta, error) {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return "", ports.RegistroReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	aleatorio := make([]byte, bytesAleatoriosReciboCargaDirecta)
	defer borrarBytesCargaDirecta(aleatorio)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", ports.RegistroReciboCargaDirecta{}, ErrEntropiaCargaDirectaNoDisponible
	}
	vinculo := calcularVinculoReciboCargaDirecta(
		claveVinculoRecibo, contexto, sesionRef, vinculoSolicitudSHA256, expiraEn, aleatorio,
	)
	defer borrarBytesCargaDirecta(vinculo)
	materialRecibo := strings.Join([]string{
		versionReciboCargaDirecta,
		claveVinculoRecibo.identificador,
		base64.RawURLEncoding.EncodeToString(aleatorio),
		vinculoSolicitudSHA256,
		strconv.FormatInt(expiraEn.UnixNano(), 10),
		hex.EncodeToString(vinculo),
	}, ".")
	indice := calcularHMACCargaDirecta(claveIndiceRecibo, "indice-recibo-carga-directa-v1", materialRecibo)
	defer borrarBytesCargaDirecta(indice)
	grupo := calcularGrupoReciboCargaDirecta(claveIndiceRecibo, contexto, sesionRef)
	defer borrarBytesCargaDirecta(grupo)
	evidenciaAltaRef, err := nuevaReferenciaOpacaCargaDirecta("recibo-alta-v1")
	if err != nil {
		return "", ports.RegistroReciboCargaDirecta{}, err
	}
	registro := ports.RegistroReciboCargaDirecta{
		IndiceHMAC:             representarHMACCargaDirecta(claveIndiceRecibo.identificador, indice),
		GrupoHMAC:              representarHMACCargaDirecta(claveIndiceRecibo.identificador, grupo),
		VinculoHMAC:            representarHMACCargaDirecta(claveVinculoRecibo.identificador, vinculo),
		EvidenciaAltaRef:       evidenciaAltaRef,
		AutorizacionEmisionRef: proyeccion.AutorizacionRef,
		ExpiraEn:               expiraEn,
	}
	if registro.Validar() != nil {
		return "", ports.RegistroReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	return materialRecibo, registro, nil
}

// completarReciboTrasAltaDurable liga al secreto la fecha elegida por la
// transaccion de alta. El indice sigue derivandose del material previo y el
// repositorio coteja RegistradoEn al consumir; asi no hace falta una segunda
// escritura y el reloj del proceso nunca se convierte en evidencia.
func completarReciboTrasAltaDurable(
	claveVinculo claveHMACCargaDirecta,
	materialRecibo string,
	registro ports.RegistroReciboCargaDirecta,
	registradoEn time.Time,
) (ports.ReciboCargaDirecta, error) {
	if registro.Validar() != nil || registradoEn.IsZero() || registradoEn.Location() != time.UTC ||
		!registradoEn.Before(registro.ExpiraEn) ||
		registro.ExpiraEn.Sub(registradoEn) > duracionMaximaReciboCargaDirecta {
		return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	atestacionAlta := calcularAtestacionAltaReciboCargaDirecta(
		claveVinculo, materialRecibo, registro.IndiceHMAC, registro.GrupoHMAC, registradoEn,
	)
	defer borrarBytesCargaDirecta(atestacionAlta)
	valorRecibo := strings.Join([]string{
		materialRecibo,
		strconv.FormatInt(registradoEn.UnixNano(), 10),
		hex.EncodeToString(atestacionAlta),
	}, ".")
	recibo, err := ports.NuevoReciboCargaDirecta(valorRecibo)
	if err != nil {
		return ports.ReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoDisponible
	}
	return recibo, nil
}

func (a *AdaptadorCriptograficoCargaDirecta) Cerrar() {
	if a == nil {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cerrado {
		return
	}
	borrarBytesCargaDirecta(a.claveSeudonimizacion.material)
	borrarBytesCargaDirecta(a.claveIndiceRecibo.material)
	borrarBytesCargaDirecta(a.claveVinculoRecibo.material)
	borrarBytesCargaDirecta(a.claveAtestacion.material)
	a.claveSeudonimizacion = claveHMACCargaDirecta{}
	a.claveIndiceRecibo = claveHMACCargaDirecta{}
	a.claveVinculoRecibo = claveHMACCargaDirecta{}
	a.claveAtestacion = claveHMACCargaDirecta{}
	a.repositorio = nil
	a.reloj = nil
	a.cerrado = true
}

func (*AdaptadorCriptograficoCargaDirecta) String() string {
	return "seguridad.AdaptadorCriptograficoCargaDirecta{[MATERIAL-OCULTO]}"
}

func (*AdaptadorCriptograficoCargaDirecta) GoString() string {
	return "seguridad.AdaptadorCriptograficoCargaDirecta{[MATERIAL-OCULTO]}"
}

func (a *AdaptadorCriptograficoCargaDirecta) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, a.String())
}

func (*AdaptadorCriptograficoCargaDirecta) MarshalJSON() ([]byte, error) {
	return []byte(`{"tipo":"adaptador_criptografico_carga_directa","material":"oculto"}`), nil
}

func (*AdaptadorCriptograficoCargaDirecta) MarshalText() ([]byte, error) {
	return []byte("[ADAPTADOR-CRIPTOGRAFICO-CARGA-DIRECTA-OCULTO]"), nil
}

func (a *AdaptadorCriptograficoCargaDirecta) LogValue() slog.Value {
	return slog.StringValue(a.String())
}

type partesReciboCargaDirecta struct {
	aleatorio              []byte
	vinculoSolicitudSHA256 string
	expiraEn               time.Time
	vinculo                []byte
	materialIndice         string
	registradoEn           time.Time
	atestacionAlta         []byte
}

func (p *partesReciboCargaDirecta) borrar() {
	if p == nil {
		return
	}
	borrarBytesCargaDirecta(p.aleatorio)
	borrarBytesCargaDirecta(p.vinculo)
	borrarBytesCargaDirecta(p.atestacionAlta)
	p.aleatorio = nil
	p.vinculo = nil
	p.atestacionAlta = nil
	p.vinculoSolicitudSHA256 = ""
	p.materialIndice = ""
	p.expiraEn = time.Time{}
	p.registradoEn = time.Time{}
}

func analizarReciboCargaDirecta(valor, identificadorVinculo string) (partesReciboCargaDirecta, error) {
	partesTexto := strings.Split(valor, ".")
	if len(partesTexto) != 8 || partesTexto[0] != versionReciboCargaDirecta ||
		partesTexto[1] != identificadorVinculo || !sha256HexadecimalCargaDirectaValido(partesTexto[3]) {
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	aleatorio, err := base64.RawURLEncoding.DecodeString(partesTexto[2])
	if err != nil || len(aleatorio) != bytesAleatoriosReciboCargaDirecta ||
		base64.RawURLEncoding.EncodeToString(aleatorio) != partesTexto[2] {
		borrarBytesCargaDirecta(aleatorio)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	expiraUnixNano, err := strconv.ParseInt(partesTexto[4], 10, 64)
	if err != nil || expiraUnixNano <= 0 || strconv.FormatInt(expiraUnixNano, 10) != partesTexto[4] {
		borrarBytesCargaDirecta(aleatorio)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	vinculo, err := hex.DecodeString(partesTexto[5])
	if err != nil || len(vinculo) != sha256.Size || hex.EncodeToString(vinculo) != partesTexto[5] {
		borrarBytesCargaDirecta(aleatorio)
		borrarBytesCargaDirecta(vinculo)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	registradoUnixNano, err := strconv.ParseInt(partesTexto[6], 10, 64)
	if err != nil || registradoUnixNano <= 0 || strconv.FormatInt(registradoUnixNano, 10) != partesTexto[6] {
		borrarBytesCargaDirecta(aleatorio)
		borrarBytesCargaDirecta(vinculo)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	expiraEn := time.Unix(0, expiraUnixNano).UTC()
	registradoEn := time.Unix(0, registradoUnixNano).UTC()
	if !registradoEn.Before(expiraEn) || expiraEn.Sub(registradoEn) > duracionMaximaReciboCargaDirecta {
		borrarBytesCargaDirecta(aleatorio)
		borrarBytesCargaDirecta(vinculo)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	atestacionAlta, err := hex.DecodeString(partesTexto[7])
	if err != nil || len(atestacionAlta) != sha256.Size || hex.EncodeToString(atestacionAlta) != partesTexto[7] {
		borrarBytesCargaDirecta(aleatorio)
		borrarBytesCargaDirecta(vinculo)
		borrarBytesCargaDirecta(atestacionAlta)
		return partesReciboCargaDirecta{}, ports.ErrReciboCargaDirectaNoValido
	}
	materialIndice := strings.Join(partesTexto[:6], ".")
	return partesReciboCargaDirecta{
		aleatorio: aleatorio, vinculoSolicitudSHA256: partesTexto[3],
		expiraEn: expiraEn, vinculo: vinculo,
		materialIndice: materialIndice, registradoEn: registradoEn,
		atestacionAlta: atestacionAlta,
	}, nil
}

func calcularVinculoReciboCargaDirecta(
	clave claveHMACCargaDirecta,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef, vinculoSolicitudSHA256 string,
	expiraEn time.Time,
	aleatorio []byte,
) []byte {
	// AutorizacionRef y Accion cambian deliberadamente entre preparacion y
	// confirmacion. El resto son los invariantes exigidos por el nucleo. La
	// huella de solicitud autentica ademas toda la preparacion, incluidos su
	// autorizacion, accion, MIME, tamano, huella, idempotencia y caducidad.
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return calcularHMACCargaDirecta(
		clave,
		"vinculo-recibo-carga-directa-v1",
		proyeccion.Esquema,
		proyeccion.OperacionRef,
		proyeccion.CorrelacionRef,
		proyeccion.Finalidad,
		proyeccion.Clasificacion,
		proyeccion.CargaRef,
		proyeccion.SujetoSeudonimoHMAC,
		proyeccion.RecursoRef,
		proyeccion.ModuloID,
		proyeccion.TipoRecurso,
		proyeccion.HuellaSolicitudHMAC,
		sesionRef,
		vinculoSolicitudSHA256,
		expiraEn.UTC().Format(time.RFC3339Nano),
		base64.RawURLEncoding.EncodeToString(aleatorio),
	)
}

func calcularGrupoReciboCargaDirecta(
	clave claveHMACCargaDirecta,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
) []byte {
	// El grupo identifica exclusivamente la carga y la sesion. Asi, cualquier
	// nueva emision para ambas sustituye el recibo activo anterior aunque hayan
	// cambiado otros invariantes de la solicitud; esos invariantes siguen
	// autenticados por VinculoHMAC y nunca amplian el permiso de consumo.
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return calcularHMACCargaDirecta(
		clave,
		"grupo-recibo-carga-directa-v1",
		proyeccion.CargaRef,
		sesionRef,
	)
}

func calcularAtestacionAltaReciboCargaDirecta(
	clave claveHMACCargaDirecta,
	materialRecibo, indiceHMAC, grupoHMAC string,
	registradoEn time.Time,
) []byte {
	return calcularHMACCargaDirecta(
		clave,
		"atestacion-alta-durable-recibo-carga-directa-v1",
		materialRecibo,
		indiceHMAC,
		grupoHMAC,
		registradoEn.Format(time.RFC3339Nano),
	)
}

func calcularAtestacionConsumoCargaDirecta(
	clave claveHMACCargaDirecta,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef string,
	resultado ports.ResultadoConsumoReciboCargaDirecta,
	validaHasta time.Time,
) []byte {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return calcularHMACCargaDirecta(
		clave,
		"atestacion-consumo-recibo-carga-directa-v1",
		proyeccion.Esquema,
		proyeccion.OperacionRef,
		proyeccion.CorrelacionRef,
		proyeccion.AutorizacionRef,
		proyeccion.Finalidad,
		proyeccion.Clasificacion,
		proyeccion.AccionNegocio,
		proyeccion.AccionTecnica,
		proyeccion.CargaRef,
		proyeccion.SujetoSeudonimoHMAC,
		proyeccion.RecursoRef,
		proyeccion.ModuloID,
		proyeccion.TipoRecurso,
		proyeccion.HuellaRecursoSHA256,
		proyeccion.HuellaSolicitudHMAC,
		proyeccion.EfectoRef,
		proyeccion.HuellaPlanEfectoSHA256,
		string(proyeccion.PasoRef),
		proyeccion.HuellaDecisionSHA256,
		sesionRef,
		resultado.IndiceHMAC,
		resultado.GrupoHMAC,
		resultado.VinculoHMAC,
		resultado.EvidenciaConsumoRef,
		resultado.IntencionConfirmacionRef,
		resultado.HuellaIntencionHMAC,
		resultado.RegistradoEn.Format(time.RFC3339Nano),
		resultado.ConsumidoEn.Format(time.RFC3339Nano),
		resultado.ExpiraEn.Format(time.RFC3339Nano),
		validaHasta.Format(time.RFC3339Nano),
	)
}

func calcularHuellaIntencionCargaDirecta(
	clave claveHMACCargaDirecta,
	contexto ports.ContextoOperacionAlmacen,
	sesionRef, vinculoHMAC, indiceHMAC, grupoHMAC, intencionRef, evidenciaConsumoRef string,
	registradoEn, validaHasta time.Time,
) []byte {
	proyeccion, err := contexto.Proyeccion()
	if err != nil {
		return nil
	}
	return calcularHMACCargaDirecta(
		clave,
		"huella-intencion-confirmacion-carga-directa-v1",
		proyeccion.Esquema,
		proyeccion.OperacionRef,
		proyeccion.CorrelacionRef,
		proyeccion.AutorizacionRef,
		proyeccion.Finalidad,
		proyeccion.Clasificacion,
		proyeccion.AccionNegocio,
		proyeccion.AccionTecnica,
		proyeccion.CargaRef,
		proyeccion.SujetoSeudonimoHMAC,
		proyeccion.RecursoRef,
		proyeccion.ModuloID,
		proyeccion.TipoRecurso,
		proyeccion.HuellaRecursoSHA256,
		proyeccion.HuellaSolicitudHMAC,
		proyeccion.EfectoRef,
		proyeccion.HuellaPlanEfectoSHA256,
		string(proyeccion.PasoRef),
		proyeccion.HuellaDecisionSHA256,
		sesionRef,
		vinculoHMAC,
		indiceHMAC,
		grupoHMAC,
		intencionRef,
		evidenciaConsumoRef,
		registradoEn.Format(time.RFC3339Nano),
		validaHasta.Format(time.RFC3339Nano),
	)
}

func hmacRepresentadoCargaDirectaValido(valor, identificador string, esperado []byte) bool {
	partes := strings.Split(valor, ":")
	if len(partes) != 3 || partes[0] != "hmac-sha256" || partes[1] != identificador ||
		partes[2] != strings.ToLower(partes[2]) {
		return false
	}
	recibido, err := hex.DecodeString(partes[2])
	if err != nil || len(recibido) != sha256.Size {
		borrarBytesCargaDirecta(recibido)
		return false
	}
	defer borrarBytesCargaDirecta(recibido)
	return hmac.Equal(recibido, esperado)
}

func calcularHMACCargaDirecta(clave claveHMACCargaDirecta, dominio string, valores ...string) []byte {
	mac := hmac.New(sha256.New, clave.material)
	escribirCampoHMACCargaDirecta(mac, dominio)
	for _, valor := range valores {
		escribirCampoHMACCargaDirecta(mac, valor)
	}
	return mac.Sum(nil)
}

func escribirCampoHMACCargaDirecta(destino hash.Hash, valor string) {
	_, _ = io.WriteString(destino, strconv.Itoa(len(valor)))
	_, _ = io.WriteString(destino, ":")
	_, _ = io.WriteString(destino, valor)
	_, _ = io.WriteString(destino, "\n")
}

func representarHMACCargaDirecta(identificador string, suma []byte) string {
	return "hmac-sha256:" + identificador + ":" + hex.EncodeToString(suma)
}

func nuevaReferenciaOpacaCargaDirecta(prefijo string) (string, error) {
	aleatorio := make([]byte, bytesEvidenciaReciboCargaDirecta)
	defer borrarBytesCargaDirecta(aleatorio)
	if _, err := rand.Read(aleatorio); err != nil {
		return "", ErrEntropiaCargaDirectaNoDisponible
	}
	return prefijo + ":" + hex.EncodeToString(aleatorio), nil
}

type instantaneaCriptografiaCargaDirecta struct {
	claveSeudonimizacion claveHMACCargaDirecta
	claveIndiceRecibo    claveHMACCargaDirecta
	claveVinculoRecibo   claveHMACCargaDirecta
	claveAtestacion      claveHMACCargaDirecta
	repositorio          ports.RepositorioRecibosCargaDirecta
	reloj                ports.Reloj
}

func (a *AdaptadorCriptograficoCargaDirecta) obtenerInstantanea() (instantaneaCriptografiaCargaDirecta, error) {
	if a == nil {
		return instantaneaCriptografiaCargaDirecta{}, ErrConfiguracionCriptografiaCargaDirectaInvalida
	}
	a.mu.RLock()
	defer a.mu.RUnlock()
	if err := a.validarDisponibleBloqueado(); err != nil {
		return instantaneaCriptografiaCargaDirecta{}, err
	}
	return instantaneaCriptografiaCargaDirecta{
		claveSeudonimizacion: copiarClaveHMACCargaDirectaInterna(a.claveSeudonimizacion),
		claveIndiceRecibo:    copiarClaveHMACCargaDirectaInterna(a.claveIndiceRecibo),
		claveVinculoRecibo:   copiarClaveHMACCargaDirectaInterna(a.claveVinculoRecibo),
		claveAtestacion:      copiarClaveHMACCargaDirectaInterna(a.claveAtestacion),
		repositorio:          a.repositorio,
		reloj:                a.reloj,
	}, nil
}

func (i *instantaneaCriptografiaCargaDirecta) borrar() {
	if i == nil {
		return
	}
	borrarBytesCargaDirecta(i.claveSeudonimizacion.material)
	borrarBytesCargaDirecta(i.claveIndiceRecibo.material)
	borrarBytesCargaDirecta(i.claveVinculoRecibo.material)
	borrarBytesCargaDirecta(i.claveAtestacion.material)
	i.claveSeudonimizacion = claveHMACCargaDirecta{}
	i.claveIndiceRecibo = claveHMACCargaDirecta{}
	i.claveVinculoRecibo = claveHMACCargaDirecta{}
	i.claveAtestacion = claveHMACCargaDirecta{}
	i.repositorio = nil
	i.reloj = nil
}

func (a *AdaptadorCriptograficoCargaDirecta) validarDisponibleBloqueado() error {
	if a.cerrado {
		return ErrCriptografiaCargaDirectaCerrada
	}
	if !claveHMACCargaDirectaInternaValida(a.claveSeudonimizacion) ||
		!claveHMACCargaDirectaInternaValida(a.claveIndiceRecibo) ||
		!claveHMACCargaDirectaInternaValida(a.claveVinculoRecibo) ||
		!claveHMACCargaDirectaInternaValida(a.claveAtestacion) ||
		dependenciaCargaDirectaNula(a.repositorio) || dependenciaCargaDirectaNula(a.reloj) {
		return ErrConfiguracionCriptografiaCargaDirectaInvalida
	}
	return nil
}

func configuracionCriptografiaCargaDirectaValida(configuracion ConfiguracionCriptografiaCargaDirecta) bool {
	claves := []ConfiguracionClaveHMACCargaDirecta{
		configuracion.ClaveSeudonimizacion,
		configuracion.ClaveIndiceRecibo,
		configuracion.ClaveVinculoRecibo,
		configuracion.ClaveAtestacion,
	}
	for indice, clave := range claves {
		if !configuracionClaveHMACCargaDirectaValida(clave) {
			return false
		}
		for anterior := 0; anterior < indice; anterior++ {
			if clave.Identificador == claves[anterior].Identificador ||
				hmac.Equal(clave.Material, claves[anterior].Material) {
				return false
			}
		}
	}
	return true
}

func configuracionClaveHMACCargaDirectaValida(clave ConfiguracionClaveHMACCargaDirecta) bool {
	return identificadorClaveHMACCargaDirectaValido(clave.Identificador) &&
		len(clave.Material) >= longitudMinimaClaveHMACCargaDirecta
}

func claveHMACCargaDirectaInternaValida(clave claveHMACCargaDirecta) bool {
	return identificadorClaveHMACCargaDirectaValido(clave.identificador) &&
		len(clave.material) >= longitudMinimaClaveHMACCargaDirecta
}

func identificadorClaveHMACCargaDirectaValido(identificador string) bool {
	if identificador == "" || identificador != strings.TrimSpace(identificador) || len(identificador) > 64 ||
		identificador[0] < 'a' || identificador[0] > 'z' {
		return false
	}
	for _, caracter := range identificador[1:] {
		if (caracter >= 'a' && caracter <= 'z') || (caracter >= '0' && caracter <= '9') ||
			caracter == '.' || caracter == '_' || caracter == '-' {
			continue
		}
		return false
	}
	return true
}

func copiarClaveHMACCargaDirecta(clave ConfiguracionClaveHMACCargaDirecta) claveHMACCargaDirecta {
	return claveHMACCargaDirecta{
		identificador: clave.Identificador,
		material:      append([]byte(nil), clave.Material...),
	}
}

func copiarClaveHMACCargaDirectaInterna(clave claveHMACCargaDirecta) claveHMACCargaDirecta {
	return claveHMACCargaDirecta{
		identificador: clave.identificador,
		material:      append([]byte(nil), clave.material...),
	}
}

func sha256HexadecimalCargaDirectaValido(valor string) bool {
	if len(valor) != sha256.Size*2 || valor != strings.ToLower(valor) || valor != strings.TrimSpace(valor) {
		return false
	}
	decodificado, err := hex.DecodeString(valor)
	borrarBytesCargaDirecta(decodificado)
	return err == nil && len(decodificado) == sha256.Size
}

func referenciaCanonicaCargaDirectaValida(valor string, maximo int) bool {
	if valor == "" || len(valor) > maximo || valor != strings.TrimSpace(valor) || !utf8.ValidString(valor) {
		return false
	}
	for _, caracter := range valor {
		if unicode.IsSpace(caracter) || unicode.IsControl(caracter) {
			return false
		}
	}
	return true
}

func validarContextoCargaDirecta(ctx context.Context) error {
	if ctx == nil {
		return ErrMaterialCriptograficoCargaDirectaInvalido
	}
	return ctx.Err()
}

func erroresContextoCargaDirecta(ctx context.Context, err error) error {
	if ctx != nil && ctx.Err() != nil {
		return ctx.Err()
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	return nil
}

func dependenciaCargaDirectaNula(dependencia any) bool {
	if dependencia == nil {
		return true
	}
	valor := reflect.ValueOf(dependencia)
	switch valor.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return valor.IsNil()
	default:
		return false
	}
}

func borrarBytesCargaDirecta(datos []byte) {
	for indice := range datos {
		datos[indice] = 0
	}
}

var _ ports.SeudonimizadorSujetoAlmacen = (*AdaptadorCriptograficoCargaDirecta)(nil)
var _ ports.EmisorReciboCargaDirecta = (*AdaptadorCriptograficoCargaDirecta)(nil)
var _ ports.ConsumidorReciboCargaDirecta = (*AdaptadorCriptograficoCargaDirecta)(nil)
var _ ports.VerificadorAtestacionConsumoReciboCargaDirecta = (*AdaptadorCriptograficoCargaDirecta)(nil)
