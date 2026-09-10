package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func refCierre87(t string) string {
	h := sha256.Sum256([]byte(t))
	return "ref:" + hex.EncodeToString(h[:])
}
func instanteCierre87(t *testing.T, s string) time.Time {
	t.Helper()
	v, e := time.Parse(time.RFC3339Nano, s)
	if e != nil {
		t.Fatal(e)
	}
	return v
}
func fixtureCierre87(t *testing.T) (domain.DefinicionSeguimiento, domain.DefinicionSeguimiento, domain.Seguimiento, domain.Seguimiento, ports.SolicitudTransaccionCierreAdministrativo) {
	t.Helper()
	b, err := os.ReadFile("../seguimientoejercicio/testdata/definicion-ejercicio.json")
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		Publicacion domain.PublicacionDefinicionSeguimiento `json:"publicacion"`
	}
	if err = json.Unmarshal(b, &fuente); err != nil {
		t.Fatal(err)
	}
	original, err := domain.RestaurarDefinicionSeguimiento(fuente.Publicacion)
	if err != nil {
		t.Fatal(err)
	}
	registrada := instanteCierre87(t, "2026-09-10T13:07:06.614186Z")
	periodo := domain.IntervaloSeguimiento{Desde: instanteCierre87(t, "2027-01-01T00:00:00Z"), Hasta: instanteCierre87(t, "2027-04-01T00:00:00Z")}
	raiz, err := domain.NuevoSeguimiento(original, domain.AltaSeguimiento{Referencia: refCierre87("seguimiento"), OrganizacionRef: "organizacion:desarrollo:dipgra", ExpedienteRef: "expediente:ct:" + strings.Repeat("e", 64), RelacionRef: refCierre87("relacion"), PeriodoPrevisto: periodo, CreadoEn: registrada.Add(-time.Second)})
	if err != nil {
		t.Fatal(err)
	}
	previo, err := raiz.Aplicar(original, 0, domain.DatosTransicionSeguimiento{ActuacionRef: refCierre87("incorporacion"), TransicionClave: "confirmar_incorporacion", MotivoClave: "ejercicio_incorporacion", ActorRef: refCierre87("actor"), UnidadRef: "unidad:desarrollo:rrhh", EfectivoEn: periodo.Desde, RegistradaEn: registrada, Periodo: &periodo, Documentos: []domain.DocumentoSeguimiento{{TipoClave: "resolucion_ejercicio", Referencia: refCierre87("resolucion")}}, ReciboRef: refCierre87("recibo_incorporacion"), CorrelacionRef: refCierre87("correlacion_incorporacion")})
	if err != nil {
		t.Fatal(err)
	}
	p := original.Publicacion()
	fecha := instanteCierre87(t, "2026-09-10T15:00:00Z")
	p.Estados = append(p.Estados, domain.EstadoDefinidoSeguimiento{Clave: domain.EstadoCerradoAdministrativamenteSeguimiento, Final: true})
	p.Motivos = append(p.Motivos, "cierre_administrativo_ejercicio")
	p.Transiciones = append(p.Transiciones, domain.TransicionDefinidaSeguimiento{Clave: domain.TransicionCerrarAdministrativamenteSinCese, Origen: "vigente", Destino: domain.EstadoCerradoAdministrativamenteSeguimiento, Clase: domain.TransicionOrdinaria, MotivoObligatorio: true, MotivosPermitidos: []domain.ClaveCatalogo{"cierre_administrativo_ejercicio"}, EfectoPeriodo: domain.EfectoPeriodoNinguno})
	sucesora, err := domain.PublicarDefinicionSeguimiento(domain.BorradorDefinicionSeguimiento{Referencia: p.Referencia, Version: 2, PublicadoEn: fecha, Vigencia: domain.VigenciaSeguimiento{Desde: fecha, Hasta: p.Vigencia.Hasta}, EstadoInicial: p.EstadoInicial, ProhibeCiclosSilenciosos: p.ProhibeCiclosSilenciosos, Estados: p.Estados, Motivos: p.Motivos, Transiciones: p.Transiciones})
	if err != nil {
		t.Fatal(err)
	}
	fecha = fecha.Add(time.Hour)
	datos := domain.DatosTransicionSeguimiento{ActuacionRef: refCierre87("cierre"), TransicionClave: domain.TransicionCerrarAdministrativamenteSinCese, MotivoClave: "cierre_administrativo_ejercicio", ActorRef: refCierre87("actor"), UnidadRef: "unidad:desarrollo:rrhh", EfectivoEn: fecha, RegistradaEn: fecha, ReciboRef: refCierre87("recibo_cierre"), CorrelacionRef: refCierre87("correlacion_cierre")}
	continuacion, err := domain.NuevaContinuacionSeguimiento(original, sucesora, previo, datos)
	if err != nil {
		t.Fatal(err)
	}
	cerrado, err := domain.AplicarCierreConContinuacion(previo, original, sucesora, continuacion, 1, datos)
	if err != nil {
		t.Fatal(err)
	}
	s := ports.SolicitudTransaccionCierreAdministrativo{Operacion: ports.OperacionCerrarAdministrativamenteSinCese, OrganizacionRef: previo.Estado().OrganizacionRef, ExpedienteRef: previo.Estado().ExpedienteRef, SeguimientoRef: previo.Estado().Referencia, VersionEsperada: 1, ClaveIdempotencia: "87000000-0000-4000-8000-000000000001", TransicionClave: datos.TransicionClave, MotivoClave: datos.MotivoClave}
	return original, sucesora, previo, cerrado, s
}

func TestCierreAdministrativoCT87ConservaPeriodo2027AlCerrarEn2026(t *testing.T) {
	original, sucesora, previo, cerrado, _ := fixtureCierre87(t)
	if !previo.Actuaciones()[0].EfectivoEn.After(cerrado.Actuaciones()[1].EfectivoEn) {
		t.Fatal("fixture no conserva efectividad futura")
	}
	if !reflect.DeepEqual(previo.PeriodosResultantes(), cerrado.PeriodosResultantes()) || previo.Estado().PeriodoPrevisto != cerrado.Estado().PeriodoPrevisto || cerrado.CeseEfectivo() != nil {
		t.Fatal("se altero periodo futuro o se creo cese")
	}
	snapshot, err := PrepararSnapshotCierreAdministrativo(original, sucesora, cerrado)
	if err != nil {
		t.Fatal(err)
	}
	var e domain.EstadoPersistidoSeguimiento
	if err = json.Unmarshal(snapshot.EstadoJSON, &e); err != nil {
		t.Fatal(err)
	}
	restaurado, err := restaurarSnapshotCierreAdministrativo(original, sucesora, e, hex.EncodeToString(snapshot.EstadoCanonico), snapshot.HuellaCanonicaSHA256)
	if err != nil || !reflect.DeepEqual(restaurado.Estado(), cerrado.Estado()) {
		t.Fatalf("snapshot posterior no recuperable: %v", err)
	}
	if _, err := PrepararSnapshotSeguimientoPersistido(original, cerrado); err == nil {
		t.Fatal("codec V1 sello continuacion")
	}
	e.PeriodosResultantes[0].Intervalo.Hasta = e.PeriodosResultantes[0].Intervalo.Hasta.Add(time.Hour)
	if _, err := restaurarSnapshotCierreAdministrativo(original, sucesora, e, hex.EncodeToString(snapshot.EstadoCanonico), snapshot.HuellaCanonicaSHA256); err == nil {
		t.Fatal("snapshot adulterado aceptado")
	}
}

func fixtureInventarioCierre87(t *testing.T) ports.EntradaInventarioTareasCierreAdministrativoEjercicio {
	t.Helper()
	_, _, previo, _, s := fixtureCierre87(t)
	raiz := previo.Estado().HuellaRaizSHA256
	libro := domain.PublicacionLibroTareasCierreAdministrativoEjercicio{Referencia: refCierre87("libro"), Version: 1, Ambito: domain.AmbitoLibroTareasCierreAdministrativoEjercicio, Tareas: []domain.TareaLibroCierreAdministrativoEjercicio{{Clave: domain.TareaIncorporacionOriginalAcreditada, Evidencia: domain.EvidenciaIncorporacionOriginal}, {Clave: domain.TareaPrimeraAnotacionAcreditada, Evidencia: domain.EvidenciaPrimeraAnotacion}}}
	estados := []domain.EstadoTareaCierreAdministrativoEjercicio{
		{Clave: domain.TareaIncorporacionOriginalAcreditada, Evidencia: domain.EvidenciaTareaCierreAdministrativoEjercicio{Tipo: domain.EvidenciaIncorporacionOriginal, Referencia: refCierre87("recibo_incorporacion"), OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, VersionSeguimientoOriginal: 1, HuellaRaizSeguimientoSHA256: raiz, VersionEvidencia: 1, HuellaEvidenciaSHA256: strings.Repeat("a", 64)}},
		{Clave: domain.TareaPrimeraAnotacionAcreditada, Evidencia: domain.EvidenciaTareaCierreAdministrativoEjercicio{Tipo: domain.EvidenciaPrimeraAnotacion, Referencia: "recibo:87000000-0000-4000-8000-000000000002", OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, VersionSeguimientoOriginal: 1, HuellaRaizSeguimientoSHA256: raiz, VersionEvidencia: 9, HuellaEvidenciaSHA256: strings.Repeat("b", 64)}},
	}
	return ports.EntradaInventarioTareasCierreAdministrativoEjercicio{Libro: libro, OrganizacionRef: s.OrganizacionRef, ExpedienteRef: s.ExpedienteRef, SeguimientoRef: s.SeguimientoRef, Estados: estados}
}

func TestCierreAdministrativoCT87InventarioLigaAmbasEvidenciasSinAlias(t *testing.T) {
	entrada := fixtureInventarioCierre87(t)
	canon, err := canonInventarioTareasCierre87(entrada)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(canon)
	inventario, err := restaurarInventarioTareasCierre87(entrada, hex.EncodeToString(canon), hex.EncodeToString(h[:]))
	if err != nil || inventario.Total != 2 || inventario.Pendientes != 0 || !inventario.Completo {
		t.Fatalf("inventario: %v", err)
	}
	if !bytes.Contains(canon, []byte(entrada.Estados[1].Evidencia.Referencia)) {
		t.Fatal("codec invento alias del recibo CT86")
	}
	entrada.Estados[1].Evidencia.VersionEvidencia++
	if _, err := restaurarInventarioTareasCierre87(entrada, hex.EncodeToString(canon), hex.EncodeToString(h[:])); err == nil {
		t.Fatal("no se detecto cambio de version de evidencia")
	}
	entrada.Estados = entrada.Estados[:1]
	if _, err := canonInventarioTareasCierre87(entrada); err == nil {
		t.Fatal("canon acepto inventario incompleto")
	}
}

type poolCierre87Prueba struct{ inicios int }

func (p *poolCierre87Prueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	p.inicios++
	return nil, errors.New("no debe abrirse")
}

type proveedorCierre87Prueba struct{ llamadas int }

func (p *proveedorCierre87Prueba) AutorizarCierreAdministrativo(context.Context, ports.SolicitudTransaccionCierreAdministrativo) (AutorizacionCierreAdministrativo, error) {
	p.llamadas++
	return AutorizacionCierreAdministrativo{}, ports.ErrCierreAdministrativoDenegado
}
func TestCierreAdministrativoCT87DeniegaAntesDeAbrirTransaccion(t *testing.T) {
	_, _, _, _, s := fixtureCierre87(t)
	pool := &poolCierre87Prueba{}
	proveedor := &proveedorCierre87Prueba{}
	tx := &TransaccionCierreAdministrativoPostgreSQL{pool: pool, proveedor: proveedor}
	callbacks := 0
	_, err := tx.EjecutarCierreAdministrativo(context.Background(), s, func(ports.PreparacionTransaccionCierreAdministrativo) (domain.Seguimiento, error) {
		callbacks++
		return domain.Seguimiento{}, nil
	})
	if !errors.Is(err, ports.ErrCierreAdministrativoDenegado) || pool.inicios != 0 || callbacks != 0 || proveedor.llamadas != 1 {
		t.Fatal("denegacion cruzo frontera")
	}
	s.Operacion = ports.OperacionCerrarAdministrativamente
	_, err = tx.EjecutarCierreAdministrativo(context.Background(), s, func(ports.PreparacionTransaccionCierreAdministrativo) (domain.Seguimiento, error) {
		callbacks++
		return domain.Seguimiento{}, nil
	})
	if err == nil || proveedor.llamadas != 1 {
		t.Fatal("adapter acepto cierre legacy con cese")
	}
}

func TestCierreAdministrativoCT87DecoderRechazaDuplicadosYCamposAjenos(t *testing.T) {
	for _, b := range []string{`{"version_resultante":2,"version_resultante":3}`, `{"version_resultante":2,"ajeno":true}`} {
		var r reciboCierreSQL87
		if decodificarCierre87(context.Background(), []byte(b), &r) == nil {
			t.Fatal("decoder acepto JSON ambiguo")
		}
	}
}

// Vector sintético transportable para comparar Go con el codec SQL CT87.
// No ejecuta ni instala SQL; la comprobación dinámica vive en pruebas_sql.
func TestCierreAdministrativoCT87VectorCanonico(t *testing.T) {
	o, p, _, cerrado, _ := fixtureCierre87(t)
	inv := fixtureInventarioCierre87(t)
	estado, err := PrepararSnapshotCierreAdministrativo(o, p, cerrado)
	if err != nil {
		t.Fatal(err)
	}
	inventario, err := canonInventarioTareasCierre87(inv)
	if err != nil {
		t.Fatal(err)
	}
	vector := struct {
		Original           domain.PublicacionDefinicionSeguimiento
		Sucesora           domain.PublicacionDefinicionSeguimiento
		Estado             domain.EstadoPersistidoSeguimiento
		CanonHex           string
		Inventario         ports.EntradaInventarioTareasCierreAdministrativoEjercicio
		InventarioCanonHex string
	}{o.Publicacion(), p.Publicacion(), cerrado.Estado(), hex.EncodeToString(estado.EstadoCanonico), inv, hex.EncodeToString(inventario)}
	b, err := json.Marshal(vector)
	if err != nil {
		t.Fatal(err)
	}
	prueba, err := os.ReadFile("../../../../../deploy/postgresql/contratacion_temporal/pruebas_sql/ct87_cierre_administrativo_sin_cese.sql")
	if err != nil {
		t.Fatal(err)
	}
	partes := bytes.Split(prueba, []byte("$vector$"))
	if len(partes) != 3 || !bytes.Equal(b, partes[1]) {
		t.Fatal("vector SQL dejó de corresponder al contrato Go")
	}
}
