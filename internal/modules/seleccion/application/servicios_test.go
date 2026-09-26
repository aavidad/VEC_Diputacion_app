package application

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"

	cose "github.com/veraison/go-cose"

	"vec-diputacion-granada/internal/modules/seleccion/domain"
	sel "vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	app "vec-diputacion-granada/internal/vec/application"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type firmantePrueba struct {
	privada ed25519.PrivateKey
	ahora   time.Time
}

func (f firmantePrueba) FirmarAtestacionAutorizacionV3(_ context.Context, s ports.SolicitudFirmaAtestacionAutorizacionV3) (ports.ResultadoFirmaAtestacionAutorizacionV3, error) {
	b, err := s.Mensaje()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	aad, err := confianza.AADExternoAtestacionAutorizacionV3("vec/prueba/seleccion")
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	m := cose.NewSign1Message()
	m.Headers.Protected.SetAlgorithm(cose.AlgorithmEdDSA)
	m.Headers.Protected[cose.HeaderLabelKeyID] = []byte("clave:prueba:seleccion")
	m.Payload = b
	signer, err := cose.NewSigner(cose.AlgorithmEdDSA, f.privada)
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	if err = m.Sign(rand.Reader, aad, signer); err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	m.Payload = nil
	m.Headers.RawProtected = nil
	m.Headers.RawUnprotected = nil
	sobre, err := m.MarshalCBOR()
	if err != nil {
		return ports.ResultadoFirmaAtestacionAutorizacionV3{}, err
	}
	return ports.NuevoResultadoFirmaAtestacionAutorizacionV3(s, sobre, "evidencia:prueba:seleccion", f.ahora)
}

// emisorPorAudiencia emite material real con la clave de capacidad de la
// audiencia de cada acción (PDP real del entorno).
type emisorPorAudiencia struct {
	emisores map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3
	llamadas int
}

func (e *emisorPorAudiencia) EmitirMaterialAutorizacionAtestadaV3(ctx context.Context, s vecdomain.SolicitudAutorizacionLigadaV3, r vecdomain.ResultadoContextoActorRegistradoV2) (vecdomain.DecisionAutorizacionLigadaV3, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3, ports.ExportadorMaterialConsumoAutorizacionAtestadaV3, error) {
	e.llamadas++
	datos, err := s.Datos()
	if err != nil {
		return vecdomain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, err
	}
	emisor := e.emisores[datos.Accion]
	if emisor == nil {
		return vecdomain.DecisionAutorizacionLigadaV3{}, ports.ConfirmacionRegistroConcesionAutorizacionLigadaV3{}, nil, vecdomain.ErrAutorizacionDenegada
	}
	return emisor.EmitirMaterialAutorizacionAtestadaV3(ctx, s, r)
}

func nuevoEmisorPrueba(t *testing.T, e *entornoAutorizacionSolicitudV3Prueba) *emisorPorAudiencia {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	exigir(t, err)
	t.Cleanup(func() { clear(priv) })
	reloj := &relojAutorizacionServicioPrueba{ahora: e.ahora}
	at, err := app.NuevoServicioAtestacionesAutorizacionV3(vecdomain.CabeceraAtestacionAutorizacionV3{FormatoVersion: vecdomain.VersionFormatoAtestacionAutorizacionV3,
		Suite: confianza.SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: "clave:prueba:seleccion", Audiencia: "vec/prueba/seleccion"}, firmantePrueba{privada: priv, ahora: e.ahora})
	exigir(t, err)
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA("clave:prueba:seleccion", 1, pub, "vec/prueba/seleccion", confianza.EstadoClaveAtestacionAutorizacionV3Activa, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{})
	exigir(t, err)
	cfg, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3("confianza:prueba:seleccion", 1, e.ahora.Add(-time.Minute), e.ahora.Add(time.Hour), raiz)
	exigir(t, err)
	ver, err := confianza.NuevoServicioConfianzaAtestacionAutorizacionV3(cfg, reloj)
	exigir(t, err)
	resultado := &emisorPorAudiencia{emisores: map[string]*confianza.EmisorMaterialAutorizacionAtestadaV3{}}
	for i, par := range append(sel.AccionesPropias(), sel.AccionesRRHH()...) {
		clave, err := confianza.NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:seleccion:"+string(rune('a'+i)), 1, bytes.Repeat([]byte{byte(0x61 + i)}, 32),
			"emisor:prueba:seleccion", par[1], confianza.EstadoClaveHMACCapacidadAtestacionV3Emision, e.ahora.Add(-time.Hour), e.ahora.Add(time.Hour), time.Time{}, 1, strings.Repeat("7", 64))
		exigir(t, err)
		capacidades, err := confianza.NuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, reloj)
		exigir(t, err)
		emisor, err := confianza.NuevoEmisorMaterialAutorizacionAtestadaV3(e.servicio, at, ver, capacidades)
		exigir(t, err)
		resultado.emisores[par[0]] = emisor
	}
	return resultado
}

func exigir(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

// Dobles en memoria de los puertos de persistencia y protección.
type repositorioPrueba struct {
	guardado   *sel.ComandoGuardarBorrador
	presentado *sel.ComandoPresentar
	llamadas   int
	version    sel.VersionSolicitud
	filas      []sel.FilaSolicitudRRHH
	ficha      sel.FichaSolicitudRRHH
	err        error
}

func (r *repositorioPrueba) GuardarBorrador(_ context.Context, c sel.ComandoGuardarBorrador) (sel.ResultadoGuardado, error) {
	r.llamadas++
	r.guardado = &c
	return sel.ResultadoGuardado{SolicitudRef: c.SolicitudRefNueva, Version: c.VersionEsperada + 1, Puntuacion: c.Puntuacion, DatosCompletos: c.DatosCompletos}, r.err
}

func (r *repositorioPrueba) Presentar(_ context.Context, c sel.ComandoPresentar) (sel.ResultadoPresentacion, error) {
	r.llamadas++
	r.presentado = &c
	return sel.ResultadoPresentacion{SolicitudRef: c.SolicitudRef, NumeroJustificante: "2026/SOL-000001", ReciboRef: c.ReciboRef, PresentadaEn: time.Now().UTC()}, r.err
}

func (r *repositorioPrueba) ListarPropias(context.Context, string, sel.MaterialConsumoV3) ([]sel.ResumenSolicitudPropia, error) {
	r.llamadas++
	return nil, r.err
}

func (r *repositorioPrueba) LeerBorrador(context.Context, string, string, sel.MaterialConsumoV3) (sel.VersionSolicitud, error) {
	r.llamadas++
	return r.version, r.err
}

func (r *repositorioPrueba) ListarPresentadas(_ context.Context, _ string, _ int64, _ int, m sel.MaterialConsumoV3) ([]sel.FilaSolicitudRRHH, error) {
	r.llamadas++
	return r.filas, r.err
}

func (r *repositorioPrueba) LeerFicha(context.Context, string, sel.MaterialConsumoV3) (sel.FichaSolicitudRRHH, error) {
	r.llamadas++
	return r.ficha, r.err
}

type registroPrueba struct {
	vigentes  []domain.ConvocatoriaPublicada
	publicado map[string]string
}

func (r *registroPrueba) PublicarConvocatoria(_ context.Context, c domain.Convocatoria) (int, bool, error) {
	h, err := c.HuellaSHA256()
	if err != nil {
		return 0, false, err
	}
	if r.publicado == nil {
		r.publicado = map[string]string{}
	}
	if r.publicado[c.Ref] == h {
		return 1, false, nil
	}
	r.publicado[c.Ref] = h
	return 1, true, nil
}

func (r *registroPrueba) ConvocatoriasVigentes(context.Context) ([]domain.ConvocatoriaPublicada, error) {
	return r.vigentes, nil
}

// protectorPrueba invierte los bytes y liga el sobre a la asociación con un
// nonce de 12 bytes derivado de ella.
type protectorPrueba struct{}

func nonceAsociacionPrueba(a sel.AsociacionDatos) []byte {
	suma := sha256.Sum256([]byte(a.PersonaRef + "|" + a.ConvocatoriaRef + "|" + strconv.Itoa(a.Version)))
	return suma[:12]
}

func (protectorPrueba) CifrarDatosSolicitud(_ context.Context, a sel.AsociacionDatos, claro []byte) (sel.SobreDatos, error) {
	cifrado := make([]byte, len(claro))
	for i, b := range claro {
		cifrado[len(claro)-1-i] = b ^ 0x5a
	}
	return sel.SobreDatos{ClaveRef: "clave:prueba", Nonce: nonceAsociacionPrueba(a), Cifrado: cifrado}, nil
}

func (protectorPrueba) ConDatosSolicitudDescifrados(_ context.Context, a sel.AsociacionDatos, s sel.SobreDatos, usar func([]byte) error) error {
	if !bytes.Equal(s.Nonce, nonceAsociacionPrueba(a)) {
		return errors.New("asociación distinta")
	}
	claro := make([]byte, len(s.Cifrado))
	for i, b := range s.Cifrado {
		claro[len(s.Cifrado)-1-i] = b ^ 0x5a
	}
	return usar(claro)
}

func (protectorPrueba) HuellaConClave(_ context.Context, dominio string, datos []byte) (string, error) {
	suma := sha256.Sum256(append([]byte(dominio+"\x00"), datos...))
	return hex.EncodeToString(suma[:]), nil
}

type referenciasPrueba struct{}

func (referenciasPrueba) NuevaReferenciaSolicitud(context.Context) (string, error) {
	return "sol_" + strings.Repeat("n", 30), nil
}

type externosNoDisponibles struct{}

func (externosNoDisponibles) FirmarPresentacion(context.Context, sel.PresentacionExterna) error {
	return sel.ErrServicioExternoNoDisponible
}
func (externosNoDisponibles) RegistrarPresentacion(context.Context, sel.PresentacionExterna) error {
	return sel.ErrServicioExternoNoDisponible
}
func (externosNoDisponibles) ComprobarTasa(context.Context, sel.PresentacionExterna) error {
	return sel.ErrServicioExternoNoDisponible
}
func (externosNoDisponibles) NotificarPresentacion(context.Context, sel.PresentacionExterna) error {
	return errors.New("proveedor caído")
}

func puntos(t *testing.T, s string) baremacion.Puntos {
	t.Helper()
	p, err := domain.ParsearPuntos(s)
	exigir(t, err)
	return p
}

func convocatoriaPrueba(t *testing.T, ahora time.Time) domain.ConvocatoriaPublicada {
	t.Helper()
	return domain.ConvocatoriaPublicada{Version: 2, Abierta: true, Convocatoria: domain.Convocatoria{
		Ref: "bolsa-operario-diputacion-2026", Titulo: "Bolsa de Operario", AbreEn: ahora.Add(-24 * time.Hour), CierraEn: ahora.Add(24 * time.Hour),
		PublicadaEn: ahora.Add(-48 * time.Hour), FechaReferencia: "2026-10-30", Turnos: []domain.Turno{{Clave: "libre", Etiqueta: "Libre"}},
		Requisitos: []domain.Requisito{{Clave: "nacionalidad", Titulo: "Nacionalidad", Obligatorio: true, ImpidePresentar: true}},
		Baremo: domain.Baremo{Maximo: puntos(t, "10"), Redondeo: baremacion.RedondeoMitadArriba, Grupos: []domain.GrupoBaremo{{Clave: "experiencia", Titulo: "Experiencia",
			Maximo: puntos(t, "6"), Meritos: []domain.MeritoBaremo{{Clave: "meses", Titulo: "Meses trabajados", Unidad: "mes", PuntosPorUnidad: puntos(t, "0.1"), Maximo: puntos(t, "6")}}}}},
		Numeracion: domain.Numeracion{Patron: "{anio}/SOL-{numero}", Ancho: 6}, MarcaEjemplo: true,
	}}
}

type entornoSeleccion struct {
	*entornoAutorizacionSolicitudV3Prueba
	orden       Orden
	emisor      *emisorPorAudiencia
	repositorio *repositorioPrueba
	registro    *registroPrueba
	propias     *ServicioSolicitudesPropias
	rrhh        *ServicioConsultaRRHH
}

func nuevoEntornoSeleccion(t *testing.T, superficie vecdomain.SuperficieAutenticacionActorV1) *entornoSeleccion {
	t.Helper()
	base := nuevoEntornoAutorizacionSolicitudV3Prueba(t, superficie)
	datos, err := base.solicitud.Datos()
	exigir(t, err)
	e := &entornoSeleccion{entornoAutorizacionSolicitudV3Prueba: base, emisor: nuevoEmisorPrueba(t, base), repositorio: &repositorioPrueba{},
		registro: &registroPrueba{vigentes: []domain.ConvocatoriaPublicada{convocatoriaPrueba(t, base.ahora)}}}
	e.orden = Orden{ResultadoContexto: base.resultado, Vinculo: datos.VinculoAutenticacionActor, Motivo: datos.ReferenciaMotivo, Correlacion: datos.Correlacion,
		Ambitos: map[string]string{"ambito_ref": "seleccion"}}
	reloj := &relojAutorizacionServicioPrueba{ahora: base.ahora}
	externos := ServiciosExternos{Firma: externosNoDisponibles{}, Registro: externosNoDisponibles{}, Tasas: externosNoDisponibles{}, Notificacion: externosNoDisponibles{}}
	e.propias, err = NuevoServicioSolicitudesPropias(e.repositorio, e.registro, e.emisor, protectorPrueba{}, referenciasPrueba{}, externos, reloj)
	exigir(t, err)
	e.rrhh, err = NuevoServicioConsultaRRHH(e.repositorio, e.registro, e.emisor, protectorPrueba{}, reloj)
	exigir(t, err)
	return e
}

func borradorCompleto() domain.Borrador {
	return domain.Borrador{Turno: "libre", Datos: domain.DatosPersonales{Nombre: "Antonio", Apellidos: "Reyes Álvarez", DocumentoIdentidad: "12345678Z",
		FechaNacimiento: "1988-04-12", Nacionalidad: "española", Correo: "antonio.reyes@example.org", Telefono: "600000000",
		Direccion: domain.Direccion{Via: "C/ Real 1", CodigoPostal: "18001", Municipio: "Granada", Provincia: "Granada"}},
		Requisitos: []domain.RequisitoDeclarado{{Clave: "nacionalidad", Estado: domain.RequisitoCumple}},
		Meritos:    []domain.MeritoDeclarado{{ClaveGrupo: "experiencia", ClaveMerito: "meses", Descripcion: "Peón de mantenimiento", Cantidad: "14"}}}
}

func TestGuardarBorradorConsumeMaterialPropioYCifraLosDatos(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	r, err := e.propias.GuardarBorrador(context.Background(), e.orden, EntradaBorrador{ConvocatoriaRef: "bolsa-operario-diputacion-2026", Clave: "clave-borrador-001", Borrador: borradorCompleto()})
	exigir(t, err)
	c := e.repositorio.guardado
	persona := e.resultado.Contexto.PersonaRef
	if c == nil || c.PersonaRef != persona || c.ConvocatoriaVersion != 2 || !c.DatosCompletos || c.DocumentoParcial != "***5678*" || len(c.DocumentoHuella) != 64 ||
		domain.FormatearPuntos(c.Puntuacion) != "1.4" || r.Version != 1 || !r.DatosCompletos {
		t.Fatalf("comando inesperado: %+v", c)
	}
	if bytes.Contains(c.Sobre.Cifrado, []byte("12345678Z")) || bytes.Contains(c.Sobre.Cifrado, []byte("Antonio")) {
		t.Fatal("los datos personales viajan en claro")
	}
	m, ok := c.Material.(ports.ExportacionMaterialConsumoAutorizacionAtestadaV3)
	if !ok || m.ValidarEstructura() != nil || m.ResumenCapacidad().Operacion() != sel.AccionGuardarBorrador ||
		m.ResumenCapacidad().AudienciaConsumo() != sel.AudienciaGuardarBorrador || m.ResumenCapacidad().EfectoRef() != sel.PrefijoRecursoPropio+persona {
		t.Fatal("material inexacto")
	}
}

func TestBorradorParcialSeGuardaSinDocumentoNiCompletitud(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	_, err := e.propias.GuardarBorrador(context.Background(), e.orden, EntradaBorrador{ConvocatoriaRef: "bolsa-operario-diputacion-2026", Clave: "clave-borrador-002",
		Borrador: domain.Borrador{Requisitos: []domain.RequisitoDeclarado{{Clave: "nacionalidad", Estado: domain.RequisitoPendiente}}}})
	exigir(t, err)
	if c := e.repositorio.guardado; c.DatosCompletos || c.DocumentoHuella != "" || c.DocumentoParcial != "" || c.Turno != "" {
		t.Fatalf("un borrador parcial no puede marcarse completo: %+v", c)
	}
}

func TestBorradorInvalidoNoLlegaAlPDPNiALaBase(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	for nombre, entrada := range map[string]EntradaBorrador{
		"clave corta":          {ConvocatoriaRef: "bolsa-operario-diputacion-2026", Clave: "corta", Borrador: borradorCompleto()},
		"convocatoria ausente": {ConvocatoriaRef: "otra", Clave: "clave-borrador-003", Borrador: borradorCompleto()},
		"mérito desconocido": {ConvocatoriaRef: "bolsa-operario-diputacion-2026", Clave: "clave-borrador-004",
			Borrador: domain.Borrador{Meritos: []domain.MeritoDeclarado{{ClaveGrupo: "x", ClaveMerito: "y", Cantidad: "1"}}}},
	} {
		if _, err := e.propias.GuardarBorrador(context.Background(), e.orden, entrada); err == nil {
			t.Fatalf("%s aceptado", nombre)
		}
	}
	if e.emisor.llamadas != 0 || e.repositorio.llamadas != 0 {
		t.Fatal("un borrador inválido llegó al PDP o a la base")
	}
}

func TestPresentarExigeDeclaracionEInformaServiciosNoDisponibles(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	entrada := EntradaPresentacion{SolicitudRef: "sol_" + strings.Repeat("n", 30), VersionEsperada: 1, Clave: "clave-presenta-01"}
	if _, err := e.propias.Presentar(context.Background(), e.orden, entrada); !errors.Is(err, sel.ErrDeclaracionRequerida) || e.emisor.llamadas != 0 {
		t.Fatal("sin declaración responsable no se presenta ni se decide")
	}
	entrada.DeclaracionResponsable = true
	r, err := e.propias.Presentar(context.Background(), e.orden, entrada)
	exigir(t, err)
	if r.Servicios["firma"] != sel.ServicioNoDisponible || r.Servicios["registro_sede"] != sel.ServicioNoDisponible ||
		r.Servicios["tasas"] != sel.ServicioNoDisponible || r.Servicios["notificacion"] != sel.ServicioFallido {
		t.Fatalf("los servicios externos deben declararse no disponibles o fallidos: %v", r.Servicios)
	}
	if p := e.repositorio.presentado; p == nil || !strings.HasPrefix(p.ReciboRef, "recibo:seleccion-presentacion:") ||
		p.Material.(ports.ExportacionMaterialConsumoAutorizacionAtestadaV3).ResumenCapacidad().AudienciaConsumo() != sel.AudienciaPresentar {
		t.Fatal("presentación inexacta")
	}
}

func TestOrdenNoAptaSeDeniegaSinTocarPDPNiBase(t *testing.T) {
	for nombre, cambiar := range map[string]func(*entornoSeleccion){
		"sin ámbitos": func(e *entornoSeleccion) { e.orden.Ambitos = nil },
		"demo": func(e *entornoSeleccion) {
			e.orden.ResultadoContexto.Contexto.Principal.AuthMethod = vecdomain.AuthMethodDemo
		},
		"sin motivo": func(e *entornoSeleccion) { e.orden.Motivo = vecdomain.ReferenciaEntradaCatalogo{} },
		"caducada": func(e *entornoSeleccion) {
			e.propias.reloj = &relojAutorizacionServicioPrueba{ahora: e.ahora.Add(time.Hour)}
		},
	} {
		t.Run(nombre, func(t *testing.T) {
			e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
			cambiar(e)
			if _, err := e.propias.Listar(context.Background(), e.orden); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) || e.emisor.llamadas != 0 || e.repositorio.llamadas != 0 {
				t.Fatalf("frontera atravesada: %v", err)
			}
		})
	}
	// La superficie interna no sirve para las solicitudes propias, ni la
	// personal para RRHH.
	interna := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	if _, err := interna.propias.Listar(context.Background(), interna.orden); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("la superficie interna atendió solicitudes propias")
	}
	personal := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	if _, err := personal.rrhh.Listar(context.Background(), personal.orden, ConsultaListado{ConvocatoriaRef: "bolsa-operario-diputacion-2026", Limite: 10}); !errors.Is(err, vecdomain.ErrAutorizacionDenegada) {
		t.Fatal("la superficie personal atendió a RRHH")
	}
}

func TestLeerBorradorDescifraConSuAsociacion(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionExternaPersonalV1)
	persona := e.resultado.Contexto.PersonaRef
	claro, err := borradorCompleto().Datos.Canonico()
	exigir(t, err)
	sobre, _ := protectorPrueba{}.CifrarDatosSolicitud(context.Background(), sel.AsociacionDatos{PersonaRef: persona, ConvocatoriaRef: "bolsa-operario-diputacion-2026", Version: 3}, claro)
	e.repositorio.version = sel.VersionSolicitud{SolicitudRef: "sol_" + strings.Repeat("n", 30), PersonaRef: persona, ConvocatoriaRef: "bolsa-operario-diputacion-2026", Version: 3, Sobre: sobre}
	b, err := e.propias.LeerBorrador(context.Background(), e.orden, "bolsa-operario-diputacion-2026")
	exigir(t, err)
	if b.Borrador.Datos.DocumentoIdentidad != "12345678Z" {
		t.Fatal("no descifró los datos")
	}
	e.repositorio.version.Version = 4
	if _, err := e.propias.LeerBorrador(context.Background(), e.orden, "bolsa-operario-diputacion-2026"); !errors.Is(err, sel.ErrNoDisponible) {
		t.Fatal("un sobre de otra versión no debe descifrarse")
	}
}

func TestRRHHListaMinimizadaYFichaConTitulos(t *testing.T) {
	e := nuevoEntornoSeleccion(t, vecdomain.SuperficieAutenticacionInternaCorporativaV1)
	persona := "per_" + strings.Repeat("q", 24)
	claro, err := borradorCompleto().Datos.Canonico()
	exigir(t, err)
	sobre, _ := protectorPrueba{}.CifrarDatosSolicitud(context.Background(), sel.AsociacionDatos{PersonaRef: persona, ConvocatoriaRef: "bolsa-operario-diputacion-2026", Version: 2}, claro)
	fila := sel.FilaSolicitudRRHH{PresentacionID: 7, SolicitudRef: "sol_" + strings.Repeat("n", 30), PersonaRef: persona, ConvocatoriaRef: "bolsa-operario-diputacion-2026",
		Version: 2, NumeroJustificante: "2026/SOL-000007", Turno: "libre", Sobre: sobre, DocumentoParcial: "***5678*", PresentadaEn: e.ahora}
	e.repositorio.filas = []sel.FilaSolicitudRRHH{fila}
	p, err := e.rrhh.Listar(context.Background(), e.orden, ConsultaListado{ConvocatoriaRef: "bolsa-operario-diputacion-2026", Limite: 1})
	exigir(t, err)
	if len(p.Filas) != 1 || p.Filas[0].NombreVisible != "Reyes Álvarez, Antonio" || p.Filas[0].DocumentoParcial != "***5678*" || p.CursorSiguiente != "7" {
		t.Fatalf("listado inesperado: %+v", p)
	}
	for _, malo := range []ConsultaListado{{ConvocatoriaRef: "bolsa-operario-diputacion-2026", Limite: 0}, {ConvocatoriaRef: "bolsa-operario-diputacion-2026", Limite: 5, Cursor: "07"}} {
		if _, err := e.rrhh.Listar(context.Background(), e.orden, malo); !errors.Is(err, sel.ErrDatosNoValidos) {
			t.Fatalf("consulta inválida aceptada: %+v", malo)
		}
	}
	e.repositorio.ficha = sel.FichaSolicitudRRHH{Fila: fila, Convocatoria: convocatoriaPrueba(t, e.ahora),
		Requisitos: []domain.RequisitoDeclarado{{Clave: "nacionalidad", Estado: domain.RequisitoCumple}},
		Meritos:    borradorCompleto().Meritos}
	f, err := e.rrhh.Detalle(context.Background(), e.orden, fila.SolicitudRef)
	exigir(t, err)
	if f.Datos.Correo != "antonio.reyes@example.org" || f.Requisitos[0].Titulo != "Nacionalidad" || f.Requisitos[0].Procedencia != domain.ProcedenciaDeclaradaPersona ||
		f.Requisitos[0].FechaReferencia != "2026-10-30" || f.Meritos[0].Titulo != "Meses trabajados" || domain.FormatearPuntos(f.Meritos[0].Puntos) != "1.4" {
		t.Fatalf("ficha inesperada: %+v", f)
	}
}

type fuentePrueba struct{ c []domain.Convocatoria }

func (f fuentePrueba) Convocatorias(context.Context) ([]domain.Convocatoria, error) { return f.c, nil }

func TestPublicarConvocatoriasEsIdempotente(t *testing.T) {
	registro := &registroPrueba{}
	fuente := fuentePrueba{c: []domain.Convocatoria{convocatoriaPrueba(t, time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)).Convocatoria}}
	primera, err := PublicarConvocatorias(context.Background(), fuente, registro)
	exigir(t, err)
	segunda, err := PublicarConvocatorias(context.Background(), fuente, registro)
	exigir(t, err)
	if !primera[0].Nueva || segunda[0].Nueva {
		t.Fatal("el segundo arranque no debe publicar otra versión")
	}
	duplicada := fuentePrueba{c: append(fuente.c, fuente.c...)}
	if _, err := PublicarConvocatorias(context.Background(), duplicada, registro); err == nil {
		t.Fatal("una convocatoria repetida en el catálogo debe detener el arranque")
	}
}
