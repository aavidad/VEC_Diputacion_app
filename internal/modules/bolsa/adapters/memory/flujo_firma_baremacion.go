package memory

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"reflect"
	"sync"
	"time"

	puertosbolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
)

var (
	_ puertosbolsa.RepositorioFlujosFirmaBaremacion         = (*RepositorioFlujosFirmaBaremacion)(nil)
	_ puertosbolsa.ProtectorEstadoFlujoFirmaBaremacion      = (*ProtectorEstadoFlujoFirmaBaremacion)(nil)
	_ puertosbolsa.GeneradorReferenciasFlujoFirmaBaremacion = GeneradorReferenciasFlujoFirmaBaremacion{}
)

const maximoFlujosFirmaBaremacionMemoria = 4_096

type arrendamientoFirmaMemoria struct {
	propietarioRef   string
	secuenciaCercado uint64
	expiraEn         time.Time
	huellaTokenHMAC  []byte
}

type finalidadHMACTokenArrendamientoMemoria uint8

const (
	finalidadSellarTokenArrendamientoMemoria finalidadHMACTokenArrendamientoMemoria = iota + 1
	finalidadVerificarTokenArrendamientoMemoria
)

type operacionHMACTokenArrendamientoMemoria func(
	token puertosbolsa.TokenArrendamientoFlujoFirmaBaremacion,
	finalidad finalidadHMACTokenArrendamientoMemoria,
	huellaEsperada []byte,
) (huellaCalculada []byte, coincide bool, err error)

// RepositorioFlujosFirmaBaremacion simula CAS, fencing, sellado e indices
// unicos bajo un mutex. Es util para pruebas de reinicio de la capa de
// aplicacion, pero no es almacenamiento durable productivo.
type RepositorioFlujosFirmaBaremacion struct {
	mu sync.RWMutex

	reloj               puertosbolsa.Reloj
	verificador         puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion
	porReferencia       map[string]puertosbolsa.ExpedienteFlujoFirmaBaremacion
	referenciaPorIndice map[string]string
	arrendamientos      map[string]arrendamientoFirmaMemoria
	secuenciaPorFlujo   map[string]uint64
	// Este cierre captura una clave efimera no reflectible y solo permite sellar
	// o verificar tokens. Un adaptador productivo debe delegar esta operacion en
	// KMS/HSM o en un secreto de instancia gestionado, versionado y rotado.
	operarHMACTokenArrendamiento operacionHMACTokenArrendamientoMemoria
}

func NuevoRepositorioFlujosFirmaBaremacion(
	reloj puertosbolsa.Reloj,
	verificador puertosbolsa.VerificadorEstadoFlujoFirmaBaremacion,
) (*RepositorioFlujosFirmaBaremacion, error) {
	if interfazNula(reloj) || interfazNula(verificador) || reloj.Ahora().UTC().IsZero() {
		return nil, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	var claveHMACTokenArrendamiento [32]byte
	if _, err := io.ReadFull(rand.Reader, claveHMACTokenArrendamiento[:]); err != nil ||
		bytesFlujoFirmaNulos(claveHMACTokenArrendamiento[:]) {
		return nil, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	claveCapturada := claveHMACTokenArrendamiento
	limpiarBytesFlujoFirma(claveHMACTokenArrendamiento[:])
	operarHMAC := func(
		token puertosbolsa.TokenArrendamientoFlujoFirmaBaremacion,
		finalidad finalidadHMACTokenArrendamientoMemoria,
		huellaEsperada []byte,
	) ([]byte, bool, error) {
		if token.Validar() != nil {
			return nil, false, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
		}
		switch finalidad {
		case finalidadSellarTokenArrendamientoMemoria:
			if huellaEsperada != nil {
				return nil, false, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
			}
			huella, err := token.HuellaHMACSHA256(claveCapturada[:])
			return huella, false, err
		case finalidadVerificarTokenArrendamientoMemoria:
			return nil, token.CoincideHuellaHMACSHA256(claveCapturada[:], huellaEsperada), nil
		default:
			return nil, false, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
		}
	}
	return &RepositorioFlujosFirmaBaremacion{
		reloj: reloj, verificador: verificador,
		porReferencia:                make(map[string]puertosbolsa.ExpedienteFlujoFirmaBaremacion),
		referenciaPorIndice:          make(map[string]string),
		arrendamientos:               make(map[string]arrendamientoFirmaMemoria),
		secuenciaPorFlujo:            make(map[string]uint64),
		operarHMACTokenArrendamiento: operarHMAC,
	}, nil
}

func (*RepositorioFlujosFirmaBaremacion) String() string {
	return "[REPOSITORIO-FLUJOS-FIRMA-BAREMACION-MEMORIA-REDACTADO]"
}
func (r *RepositorioFlujosFirmaBaremacion) GoString() string { return r.String() }
func (r *RepositorioFlujosFirmaBaremacion) Format(estado fmt.State, _ rune) {
	_, _ = io.WriteString(estado, r.String())
}
func (r *RepositorioFlujosFirmaBaremacion) LogValue() slog.Value {
	return slog.StringValue(r.String())
}

func (r *RepositorioFlujosFirmaBaremacion) CrearORecuperarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudCrearORecuperarFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil || r == nil || solicitud.Expediente.Validar() != nil ||
		solicitud.Expediente.Version != 1 || len(solicitud.Expediente.PuntosControl) != 0 ||
		solicitud.Expediente.Estado != puertosbolsa.EstadoExpedienteFirmaPreparando {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	if err := r.verificarExpediente(ctx, solicitud.Expediente); err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, err
	}
	if referencia, existe := r.referenciaPorIndice[solicitud.Expediente.IndiceIdempotenciaHMAC]; existe {
		existente := r.porReferencia[referencia]
		if !mismaSolicitudInicialFlujoFirma(existente, solicitud.Expediente) {
			return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrClaveFlujoFirmaBaremacionReutilizada
		}
		clon, err := existente.Clonar()
		if err != nil {
			return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
		}
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{Expediente: clon, Creado: false}, nil
	}
	if _, colision := r.porReferencia[solicitud.Expediente.FlujoRef]; colision ||
		len(r.porReferencia) >= maximoFlujosFirmaBaremacionMemoria {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	}
	ahora := r.reloj.Ahora().UTC()
	if ahora.IsZero() || solicitud.Expediente.CreadoEn.After(ahora) || solicitud.Expediente.ActualizadoEn.After(ahora) {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	clon, err := solicitud.Expediente.Clonar()
	if err != nil {
		return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	r.porReferencia[clon.FlujoRef] = clon
	r.referenciaPorIndice[clon.IndiceIdempotenciaHMAC] = clon.FlujoRef
	salida, _ := clon.Clonar()
	return puertosbolsa.ResultadoCrearORecuperarFlujoFirmaBaremacion{Expediente: salida, Creado: true}, nil
}

func (r *RepositorioFlujosFirmaBaremacion) ObtenerFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudObtenerFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil || r == nil || solicitud.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	}
	r.mu.RLock()
	expediente, existe := r.porReferencia[solicitud.FlujoRef]
	coincide := existe && expediente.IndiceIdempotenciaHMAC == solicitud.IndiceIdempotenciaHMAC &&
		expediente.VinculoActorHMAC == solicitud.VinculoActorHMAC
	var clon puertosbolsa.ExpedienteFlujoFirmaBaremacion
	var err error
	if coincide {
		clon, err = expediente.Clonar()
	}
	r.mu.RUnlock()
	if !coincide || err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	}
	if err := r.verificarExpediente(ctx, clon); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	return clon, nil
}

func (r *RepositorioFlujosFirmaBaremacion) AdquirirArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudAdquirirArrendamientoFlujoFirmaBaremacion,
) (puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion, error) {
	if solicitud.Validar() != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	expediente, err := r.ObtenerFlujoFirmaBaremacion(ctx, solicitud.Consulta)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, err
	}
	token, err := puertosbolsa.NuevoTokenArrendamientoFlujoFirmaBaremacion()
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	huellaTokenHMAC, _, err := r.operarHMACTokenArrendamiento(
		token,
		finalidadSellarTokenArrendamientoMemoria,
		nil,
	)
	if err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, err
	}
	actual := r.porReferencia[solicitud.Consulta.FlujoRef]
	if actual.Version != solicitud.VersionEsperada || actual.Version != expediente.Version {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	}
	ahora := r.reloj.Ahora().UTC()
	if ahora.IsZero() {
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	if vigente, existe := r.arrendamientos[actual.FlujoRef]; existe {
		if ahora.Before(vigente.expiraEn) {
			return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrFlujoFirmaBaremacionOcupado
		}
		limpiarBytesFlujoFirma(vigente.huellaTokenHMAC)
	}
	r.secuenciaPorFlujo[actual.FlujoRef]++
	registro := arrendamientoFirmaMemoria{
		propietarioRef:   solicitud.PropietarioRef,
		secuenciaCercado: r.secuenciaPorFlujo[actual.FlujoRef],
		expiraEn:         ahora.Add(solicitud.Duracion),
		huellaTokenHMAC:  append([]byte(nil), huellaTokenHMAC...),
	}
	r.arrendamientos[actual.FlujoRef] = registro
	arrendamiento := puertosbolsa.ArrendamientoFlujoFirmaBaremacion{
		FlujoRef: actual.FlujoRef, PropietarioRef: registro.propietarioRef,
		SecuenciaCercado: registro.secuenciaCercado, ExpiraEn: registro.expiraEn, Token: token,
	}
	clon, err := actual.Clonar()
	if err != nil || arrendamiento.Validar() != nil {
		limpiarBytesFlujoFirma(registro.huellaTokenHMAC)
		delete(r.arrendamientos, actual.FlujoRef)
		return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{}, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	return puertosbolsa.ResultadoAdquirirArrendamientoFlujoFirmaBaremacion{
		Expediente: clon, Arrendamiento: arrendamiento,
	}, nil
}

func (r *RepositorioFlujosFirmaBaremacion) GuardarFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudGuardarFlujoFirmaBaremacion,
) (puertosbolsa.ExpedienteFlujoFirmaBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil || r == nil || solicitud.Validar() != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrSolicitudFlujoFirmaBaremacionInvalida
	}
	if err := r.verificarExpediente(ctx, solicitud.Siguiente); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, err
	}
	actual, existe := r.porReferencia[solicitud.Siguiente.FlujoRef]
	if !existe || actual.IndiceIdempotenciaHMAC != solicitud.Siguiente.IndiceIdempotenciaHMAC ||
		actual.VinculoActorHMAC != solicitud.Siguiente.VinculoActorHMAC {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrFlujoFirmaBaremacionNoEncontrado
	}
	if actual.Version != solicitud.VersionEsperada {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrConflictoFlujoFirmaBaremacion
	}
	ahora := r.reloj.Ahora().UTC()
	vigente, existe := r.arrendamientos[actual.FlujoRef]
	_, tokenCoincide, errorToken := r.operarHMACTokenArrendamiento(
		solicitud.Arrendamiento.Token,
		finalidadVerificarTokenArrendamientoMemoria,
		vigente.huellaTokenHMAC,
	)
	if !existe || ahora.IsZero() || !ahora.Before(vigente.expiraEn) ||
		vigente.propietarioRef != solicitud.Arrendamiento.PropietarioRef ||
		vigente.secuenciaCercado != solicitud.Arrendamiento.SecuenciaCercado ||
		!vigente.expiraEn.Equal(solicitud.Arrendamiento.ExpiraEn) || errorToken != nil || !tokenCoincide {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	if !transicionFlujoFirmaValida(actual, solicitud.Siguiente) || solicitud.Siguiente.ActualizadoEn.After(ahora) {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	clon, err := solicitud.Siguiente.Clonar()
	if err != nil {
		return puertosbolsa.ExpedienteFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	r.porReferencia[actual.FlujoRef] = clon
	salida, _ := clon.Clonar()
	return salida, nil
}

func (r *RepositorioFlujosFirmaBaremacion) LiberarArrendamientoFlujoFirmaBaremacion(
	ctx context.Context,
	solicitud puertosbolsa.SolicitudLiberarArrendamientoFlujoFirmaBaremacion,
) error {
	if err := validarContextoEjecucion(ctx); err != nil || r == nil || solicitud.Arrendamiento.Validar() != nil {
		return puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := validarContextoEjecucion(ctx); err != nil {
		return err
	}
	vigente, existe := r.arrendamientos[solicitud.Arrendamiento.FlujoRef]
	if !existe {
		return nil
	}
	_, tokenCoincide, errorToken := r.operarHMACTokenArrendamiento(
		solicitud.Arrendamiento.Token,
		finalidadVerificarTokenArrendamientoMemoria,
		vigente.huellaTokenHMAC,
	)
	if vigente.propietarioRef != solicitud.Arrendamiento.PropietarioRef ||
		vigente.secuenciaCercado != solicitud.Arrendamiento.SecuenciaCercado ||
		!vigente.expiraEn.Equal(solicitud.Arrendamiento.ExpiraEn) || errorToken != nil || !tokenCoincide {
		return puertosbolsa.ErrArrendamientoFlujoFirmaInvalido
	}
	limpiarBytesFlujoFirma(vigente.huellaTokenHMAC)
	delete(r.arrendamientos, solicitud.Arrendamiento.FlujoRef)
	return nil
}

func limpiarBytesFlujoFirma(valor []byte) {
	for indice := range valor {
		valor[indice] = 0
	}
}

func bytesFlujoFirmaNulos(valor []byte) bool {
	acumulado := byte(0)
	for _, elemento := range valor {
		acumulado |= elemento
	}
	return acumulado == 0
}

func (r *RepositorioFlujosFirmaBaremacion) verificarExpediente(
	ctx context.Context,
	expediente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) error {
	carga, err := puertosbolsa.RepresentacionCanonicaExpedienteFlujoFirmaBaremacion(expediente)
	if err != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	solicitud := puertosbolsa.SolicitudVerificarEstadoFlujoFirmaBaremacion{
		RepresentacionCanonica: carga, SelloHMAC: expediente.SelloEstadoHMAC,
	}
	if solicitud.Validar() != nil || r.verificador.VerificarEstadoFlujoFirmaBaremacion(ctx, solicitud) != nil {
		return puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return nil
}

func mismaSolicitudInicialFlujoFirma(
	a, b puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) bool {
	return a.IndiceIdempotenciaHMAC == b.IndiceIdempotenciaHMAC &&
		a.HuellaSolicitudHMAC == b.HuellaSolicitudHMAC && a.VinculoActorHMAC == b.VinculoActorHMAC &&
		a.PerfilActorClave == b.PerfilActorClave && a.ProcesoRef == b.ProcesoRef &&
		a.SolicitudRef == b.SolicitudRef && a.BaremacionMeritoRef == b.BaremacionMeritoRef &&
		a.DecisionRef == b.DecisionRef
}

func transicionFlujoFirmaValida(
	anterior, siguiente puertosbolsa.ExpedienteFlujoFirmaBaremacion,
) bool {
	if anterior.Validar() != nil || siguiente.Validar() != nil ||
		anterior.FlujoRef != siguiente.FlujoRef || siguiente.Version != anterior.Version+1 ||
		anterior.IndiceIdempotenciaHMAC != siguiente.IndiceIdempotenciaHMAC ||
		anterior.HuellaSolicitudHMAC != siguiente.HuellaSolicitudHMAC ||
		anterior.VinculoActorHMAC != siguiente.VinculoActorHMAC ||
		anterior.PerfilActorClave != siguiente.PerfilActorClave ||
		anterior.ProcesoRef != siguiente.ProcesoRef || anterior.SolicitudRef != siguiente.SolicitudRef ||
		anterior.BaremacionMeritoRef != siguiente.BaremacionMeritoRef || anterior.DecisionRef != siguiente.DecisionRef ||
		!anterior.CreadoEn.Equal(siguiente.CreadoEn) || siguiente.ActualizadoEn.Before(anterior.ActualizadoEn) {
		return false
	}
	if len(siguiente.PuntosControl) == len(anterior.PuntosControl)+1 {
		if !puntosControlFirmaIguales(anterior.PuntosControl, siguiente.PuntosControl[:len(anterior.PuntosControl)]) ||
			siguiente.PuntosControl[len(siguiente.PuntosControl)-1].Estado != puertosbolsa.EstadoPuntoControlFirmaDeclarado ||
			!estadosProtegidosIguales(anterior.EstadoProtegido, siguiente.EstadoProtegido) ||
			!reflect.DeepEqual(anterior.ProyeccionLanzamiento, siguiente.ProyeccionLanzamiento) ||
			!reflect.DeepEqual(anterior.Resultado, siguiente.Resultado) {
			return false
		}
		return true
	}
	if len(siguiente.PuntosControl) != len(anterior.PuntosControl) || len(anterior.PuntosControl) == 0 ||
		!puntosControlFirmaIguales(anterior.PuntosControl[:len(anterior.PuntosControl)-1], siguiente.PuntosControl[:len(siguiente.PuntosControl)-1]) {
		return false
	}
	a := anterior.PuntosControl[len(anterior.PuntosControl)-1]
	b := siguiente.PuntosControl[len(siguiente.PuntosControl)-1]
	if a.Estado != puertosbolsa.EstadoPuntoControlFirmaDeclarado ||
		b.Estado != puertosbolsa.EstadoPuntoControlFirmaCompletado || a.Paso != b.Paso ||
		a.EfectoRef != b.EfectoRef || a.ClaveIdempotenciaHMAC != b.ClaveIdempotenciaHMAC ||
		!a.DeclaradoEn.Equal(b.DeclaradoEn) {
		return false
	}
	switch b.Paso {
	case puertosbolsa.PasoPrepararFirmaBaremacion:
		return anterior.ProyeccionLanzamiento == nil && siguiente.ProyeccionLanzamiento != nil &&
			anterior.Resultado == nil && siguiente.Resultado == nil
	case puertosbolsa.PasoConfirmarFirmaBaremacion:
		return reflect.DeepEqual(anterior.ProyeccionLanzamiento, siguiente.ProyeccionLanzamiento) &&
			anterior.Resultado == nil && siguiente.Resultado != nil
	default:
		return reflect.DeepEqual(anterior.ProyeccionLanzamiento, siguiente.ProyeccionLanzamiento) &&
			reflect.DeepEqual(anterior.Resultado, siguiente.Resultado)
	}
}

func puntosControlFirmaIguales(a, b []puertosbolsa.PuntoControlFirmaBaremacion) bool {
	if len(a) != len(b) {
		return false
	}
	for indice := range a {
		if !reflect.DeepEqual(a[indice], b[indice]) {
			return false
		}
	}
	return true
}

func estadosProtegidosIguales(
	a, b puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion,
) bool {
	da, errA := a.DatosPersistencia()
	db, errB := b.DatosPersistencia()
	return errA == nil && errB == nil && da.Esquema == db.Esquema && da.Algoritmo == db.Algoritmo &&
		da.ClaveRef == db.ClaveRef && da.HuellaSHA256 == db.HuellaSHA256 &&
		bytes.Equal(da.Nonce, db.Nonce) && bytes.Equal(da.Cifrado, db.Cifrado)
}

// ProtectorEstadoFlujoFirmaBaremacion usa AES-256-GCM con AAD de esquema y
// referencia de clave. La clave se inyecta para que dos instancias puedan
// reanudar el mismo expediente; un despliegue productivo debe obtenerla de un
// KMS/HSM y aplicar rotacion/versionado.
type ProtectorEstadoFlujoFirmaBaremacion struct {
	claveRef string
	aead     cipher.AEAD
	azar     io.Reader
}

func NuevoProtectorEstadoFlujoFirmaBaremacion(
	claveRef string,
	clave []byte,
) (*ProtectorEstadoFlujoFirmaBaremacion, error) {
	if claveRef == "" || len(clave) != 32 {
		return nil, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	bloque, err := aes.NewCipher(append([]byte(nil), clave...))
	if err != nil {
		return nil, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	aead, err := cipher.NewGCM(bloque)
	if err != nil || aead.NonceSize() != 12 {
		return nil, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return &ProtectorEstadoFlujoFirmaBaremacion{claveRef: claveRef, aead: aead, azar: rand.Reader}, nil
}

func (p *ProtectorEstadoFlujoFirmaBaremacion) ProtegerEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	carga puertosbolsa.CargaProtegida,
) (puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion, error) {
	if err := validarContextoEjecucion(ctx); err != nil || p == nil || p.aead == nil || carga.Validar() != nil {
		return puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	nonce := make([]byte, p.aead.NonceSize())
	if _, err := io.ReadFull(p.azar, nonce); err != nil {
		return puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	cifrado := p.aead.Seal(nil, nonce, carga.Revelar(), aadEstadoFlujoFirma(p.claveRef))
	return puertosbolsa.NuevoEstadoProtegidoFlujoFirmaBaremacion(
		puertosbolsa.AlgoritmoProteccionEstadoAES256GCM, p.claveRef, nonce, cifrado,
	)
}

func (p *ProtectorEstadoFlujoFirmaBaremacion) DesprotegerEstadoFlujoFirmaBaremacion(
	ctx context.Context,
	estado puertosbolsa.EstadoProtegidoFlujoFirmaBaremacion,
) (puertosbolsa.CargaProtegida, error) {
	if err := validarContextoEjecucion(ctx); err != nil || p == nil || p.aead == nil || estado.Validar() != nil {
		return puertosbolsa.CargaProtegida{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	datos, err := estado.DatosPersistencia()
	if err != nil || datos.ClaveRef != p.claveRef || datos.Algoritmo != puertosbolsa.AlgoritmoProteccionEstadoAES256GCM {
		return puertosbolsa.CargaProtegida{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	plano, err := p.aead.Open(nil, datos.Nonce, datos.Cifrado, aadEstadoFlujoFirma(p.claveRef))
	if err != nil {
		return puertosbolsa.CargaProtegida{}, puertosbolsa.ErrEstadoFlujoFirmaAlterado
	}
	return puertosbolsa.NuevaCargaProtegida(plano)
}

func aadEstadoFlujoFirma(claveRef string) []byte {
	return []byte(puertosbolsa.EsquemaEstadoProtegidoFlujoFirmaBaremacion + "\x00" + claveRef)
}

type GeneradorReferenciasFlujoFirmaBaremacion struct{}

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaFlujoFirmaBaremacion() (string, error) {
	return referenciaAleatoriaFlujoFirma("flujo-firma-baremacion")
}

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaPropietarioArrendamientoFirmaBaremacion() (string, error) {
	return referenciaAleatoriaFlujoFirma("propietario-arrendamiento-firma")
}

func (GeneradorReferenciasFlujoFirmaBaremacion) NuevaReferenciaEfectoFirmaBaremacion(
	paso puertosbolsa.PasoFlujoFirmaBaremacion,
) (string, error) {
	if !paso.Valido() {
		return "", puertosbolsa.ErrPasoFlujoFirmaNoPermitido
	}
	return referenciaAleatoriaFlujoFirma("efecto-" + string(paso))
}

func referenciaAleatoriaFlujoFirma(prefijo string) (string, error) {
	bytesAleatorios := make([]byte, 24)
	if _, err := io.ReadFull(rand.Reader, bytesAleatorios); err != nil {
		return "", err
	}
	return prefijo + ":" + base64.RawURLEncoding.EncodeToString(bytesAleatorios), nil
}
