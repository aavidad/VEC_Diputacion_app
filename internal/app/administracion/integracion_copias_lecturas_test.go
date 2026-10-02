package administracion

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
	poladapter "vec-diputacion-granada/internal/modules/administracion/adapters/politicacopias"
	regadapter "vec-diputacion-granada/internal/modules/administracion/adapters/registrocopias"
	"vec-diputacion-granada/internal/modules/administracion/domain/copias"
	"vec-diputacion-granada/internal/modules/administracion/domain/operacionescopias"
	ej "vec-diputacion-granada/internal/modules/administracion/ports/ejecucioncopias"
	p "vec-diputacion-granada/internal/modules/administracion/ports/httpcopias"
	reg "vec-diputacion-granada/internal/modules/administracion/ports/registrocopias"
)

type destinoLecturasSintetico struct {
	conjunto ej.Conjunto
	err      error
	llamadas int
}

func (d *destinoLecturasSintetico) Recuperar(context.Context, string) (ej.Conjunto, error) {
	d.llamadas++
	return d.conjunto, d.err
}
func (d *destinoLecturasSintetico) Publicar(context.Context, ej.Captura) (ej.Conjunto, error) {
	return ej.Conjunto{}, p.ErrNoDisponible
}
func (d *destinoLecturasSintetico) CerrarVerificacion(context.Context, ej.Conjunto, copias.Verificacion) (ej.Conjunto, error) {
	return ej.Conjunto{}, p.ErrNoDisponible
}
func TestLectorDiarioRealMinimizaYSoloRecuperaConjuntoAutenticado(t *testing.T) {
	b, err := os.ReadFile("../../../cmd/vec-copias-comprobar/testdata/compatible.json")
	if err != nil {
		t.Fatal(err)
	}
	var ejemplo struct {
		Manifiesto copias.Manifiesto `json:"manifiesto"`
	}
	if json.Unmarshal(b, &ejemplo) != nil {
		t.Fatal("fixture")
	}
	m := ejemplo.Manifiesto
	base := t.TempDir()
	dir := filepath.Join(base, "diario")
	root := filepath.Join(base, "estado")
	if os.Mkdir(dir, 0700) != nil || os.Mkdir(root, 0700) != nil {
		t.Fatal("fixture dirs")
	}
	diario, err := regadapter.Abrir(regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100})
	if err != nil {
		t.Fatal(err)
	}
	ses := sesionPrueba(t)
	solicitud := operacionescopias.Solicitud{Operacion: m.OperacionRef, Clave: "clave:ejemplo", SHA256: strings.Repeat("a", 64), Conjunto: m.ConjuntoRef, Destino: "destino:ejemplo", Politica: m.PoliticaRef}
	reserva, err := diario.Reservar(context.Background(), declaracionCopias(ses), solicitud)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = diario.Aplicar(context.Background(), declaracionCopias(ses), solicitud.Operacion, operacionescopias.Comando{Clave: "inicio:fixture", SolicitudSHA256: solicitud.SHA256, Accion: "iniciar_captura"}); err != nil {
		t.Fatal(err)
	}
	if _, err = diario.Aplicar(context.Background(), declaracionCopias(ses), solicitud.Operacion, operacionescopias.Comando{Clave: "confirmacion:fixture", SolicitudSHA256: solicitud.SHA256, VersionEsperada: 1, Accion: "confirmar_captura", ManifiestoSHA256: huellaManifiestoADMIN(m)}); err != nil {
		t.Fatal(err)
	}
	reabierto, err := regadapter.Abrir(regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100})
	if err != nil {
		t.Fatal(err)
	}
	listado, err := reabierto.Listar(context.Background(), declaracionCopias(ses), reg.Consulta{Limite: 100})
	if err != nil || len(listado.Operaciones) != 1 || listado.Operaciones[0].Reserva != reserva.Recibo {
		t.Fatal("primera fecha alterada tras reabrir")
	}
	diario = reabierto
	destino := &destinoLecturasSintetico{conjunto: ej.Conjunto{Ref: m.ConjuntoRef, IndiceAutenticadoRef: "indice:fixture", Manifiesto: m, ManifiestoSHA256: huellaManifiestoADMIN(m)}}
	autoridad := &autoridadCopiasPrueba{recurso: m.ConjuntoRef, operacion: m.OperacionRef}
	auditor := &auditorCopiasPrueba{}
	lector := &LecturasCopias{Autoridad: autoridad, Auditor: auditor, Diario: diario, Destino: destino}
	pagina, err := lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(pagina.Copias) != 1 || pagina.Copias[0].CopiaRef != m.ConjuntoRef || pagina.Copias[0].Compatibilidad.Estado != "no_comprobable" {
		t.Fatalf("lectura: %+v %v", pagina, err)
	}
	// La declaración inicial CS07 no decide la verificación del manifiesto.
	if pagina.Copias[0].Estado != m.Verificacion.Estado {
		t.Fatal("estado inferido del diario")
	}

	// La página mezcla una copia autenticada con reservas/progreso reales. Las
	// filas prepublicación no llaman al proveedor de manifiesto ni ocultan la copia.
	observado, err := regadapter.AbrirConObservadorAbandono(regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100}, observadorAbandonoADMIN{})
	if err != nil {
		t.Fatal(err)
	}
	permisos := map[string]bool{m.ConjuntoRef: true, m.OperacionRef: true}
	for _, estado := range []string{"solicitada", "capturando", "abandono"} {
		nueva := solicitud
		nueva.Operacion = "operacion:" + estado
		nueva.Clave = "clave:" + estado
		nueva.Conjunto = "conjunto:" + estado
		nueva.Destino = "destino:" + estado
		permisos[nueva.Operacion] = true
		permisos[nueva.Conjunto] = true
		if _, err = observado.Reservar(context.Background(), declaracionCopias(ses), nueva); err != nil {
			t.Fatal(err)
		}
		if estado != "solicitada" {
			if _, err = observado.Aplicar(context.Background(), declaracionCopias(ses), nueva.Operacion, operacionescopias.Comando{Clave: "inicio:" + estado, SolicitudSHA256: nueva.SHA256, Accion: "iniciar_captura"}); err != nil {
				t.Fatal(err)
			}
		}
		if estado == "abandono" {
			if _, err = observado.AbandonarCaptura(context.Background(), declaracionCopias(ses), reg.SolicitudAbandono{Operacion: nueva.Operacion, Clave: "abandonar:" + estado, VersionEsperada: 1, SolicitudSHA256: nueva.SHA256, Destino: nueva.Destino, FalloReferencia: "captura_fallida", FalloSHA256: strings.Repeat("f", 64)}); err != nil {
				t.Fatal(err)
			}
		}
	}
	lector.Diario = observado
	autoridad.recursos = permisos
	llamadasManifiesto := destino.llamadas
	pagina, err = lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(pagina.Copias) != 4 || destino.llamadas != llamadasManifiesto+1 {
		t.Fatal("mezcla progreso bloquea copia publicada", pagina, err)
	}
	for _, c := range pagina.Copias {
		if c.CopiaRef != m.ConjuntoRef && (c.HuellaSHA256 != "" || c.ReleaseRef != "" || c.TamanoBytes != nil || c.FinalizadaEn != nil || c.Compatibilidad.Estado != "no_comprobable") {
			t.Fatal("progreso inventa metadatos")
		}
	}
	autoridad.err = p.ErrDenegado
	antes := destino.llamadas
	if _, err = lector.Detalle(context.Background(), ses, m.ConjuntoRef); !errors.Is(err, p.ErrDenegado) || destino.llamadas != antes {
		t.Fatal("sin permiso recupera contenido")
	}
	autoridad.err = nil
	destino.err = errors.New("fallo_sintetico")
	if _, err = lector.Listar(context.Background(), ses, "", 25); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("fallo autenticacion no bloquea")
	}
	destino.err = nil
	destino.conjunto.Manifiesto.OperacionRef = "operacion:ajena"
	destino.conjunto.ManifiestoSHA256 = huellaManifiestoADMIN(destino.conjunto.Manifiesto)
	if _, err = lector.Listar(context.Background(), ses, "", 25); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("conjunto de otra operacion aceptado")
	}
	politica, err := poladapter.Abrir(filepath.Join(base, "politica"))
	if err != nil {
		t.Fatal(err)
	}
	defer politica.Cerrar()
	lector.Politica = politica
	if _, err = lector.Calendario(context.Background(), ses); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("politica vacia devuelve exito")
	}
	if _, err = lector.Retencion(context.Background(), ses); !errors.Is(err, p.ErrNoDisponible) {
		t.Fatal("retencion vacia devuelve exito")
	}

}

type observadorAbandonoADMIN struct{}

func (observadorAbandonoADMIN) ConfirmarAbandono(_ context.Context, _ reg.Declaracion, s reg.SolicitudAbandono) (operacionescopias.ObservacionAbandono, error) {
	return operacionescopias.ObservacionAbandono{Operacion: s.Operacion, Destino: s.Destino, FalloReferencia: s.FalloReferencia, FalloSHA256: s.FalloSHA256, Lease: "lease:fixture", EstadoEfecto: "inactivo", EstadoLease: "cancelada", EstadoPlataforma: "sin_efectos_pendientes", EstadoVerificador: "detenido", EstadoVentana: "inactiva"}, nil
}
func TestListadoProgresoNoRecuperaIndiceInexistenteYAbandonoPublicadoNoOcultaError(t *testing.T) {
	ses := sesionPrueba(t)
	base := t.TempDir()
	dir := filepath.Join(base, "diario")
	root := filepath.Join(base, "estado")
	if os.Mkdir(dir, 0700) != nil || os.Mkdir(root, 0700) != nil {
		t.Fatal("dirs")
	}
	cfg := regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100}
	diario, err := regadapter.AbrirConObservadorAbandono(cfg, observadorAbandonoADMIN{})
	if err != nil {
		t.Fatal(err)
	}
	s := operacionescopias.Solicitud{Operacion: "operacion:pending", Clave: "clave:pending", SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:pending", Destino: "destino:pending", Politica: "politica:fixture"}
	reserva, err := diario.Reservar(context.Background(), declaracionCopias(ses), s)
	if err != nil {
		t.Fatal(err)
	}
	destino := &destinoLecturasSintetico{err: p.ErrNoEncontrado}
	lector := &LecturasCopias{Diario: diario, Destino: destino, Autoridad: &autoridadCopiasPrueba{recurso: s.Conjunto, operacion: s.Operacion}, Auditor: &auditorCopiasPrueba{}}
	v, err := lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(v.Copias) != 1 || v.Copias[0].Estado != "solicitada" || destino.llamadas != 0 {
		t.Fatal("reserva exige indice", v, err)
	}
	if v.Copias[0].IniciadaEn.Format(time.RFC3339Nano) != reserva.Recibo.Instante {
		t.Fatal("inicio fabricado")
	}
	if _, err = diario.Aplicar(context.Background(), declaracionCopias(ses), s.Operacion, operacionescopias.Comando{Clave: "inicio:pending", SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"}); err != nil {
		t.Fatal(err)
	}
	if _, err = diario.AbandonarCaptura(context.Background(), declaracionCopias(ses), reg.SolicitudAbandono{Operacion: s.Operacion, Clave: "abandonar:pending", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Destino: s.Destino, FalloReferencia: "captura_fallida", FalloSHA256: strings.Repeat("f", 64)}); err != nil {
		t.Fatal(err)
	}
	v, err = lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(v.Copias) != 1 || v.Copias[0].Estado != "fallida" || destino.llamadas != 0 {
		t.Fatal("abandono prepublicado exige indice", v, err)
	}
	diario, err = regadapter.AbrirConObservadorAbandono(cfg, observadorAbandonoADMIN{})
	if err != nil {
		t.Fatal(err)
	}
	lector.Diario = diario
	v, err = lector.Listar(context.Background(), ses, "", 25)
	if err != nil || v.Copias[0].IniciadaEn.Format(time.RFC3339Nano) != reserva.Recibo.Instante {
		t.Fatal("reapertura pierde reserva")
	}
	// Publicación anterior a la confirmación CS07: el fallo gobernado de
	// verificación exige recuperar el índice incluso al abandonar desde capturando.
	s.Operacion = "operacion:publicada"
	s.Clave = "clave:publicada"
	s.Conjunto = "conjunto:publicada"
	lector.Autoridad = &autoridadCopiasPrueba{recurso: s.Conjunto, operacion: s.Operacion}
	if _, err = diario.Reservar(context.Background(), declaracionCopias(ses), s); err != nil {
		t.Fatal(err)
	}
	if _, err = diario.Aplicar(context.Background(), declaracionCopias(ses), s.Operacion, operacionescopias.Comando{Clave: "inicio:publicada", SolicitudSHA256: s.SHA256, Accion: "iniciar_captura"}); err != nil {
		t.Fatal(err)
	}
	if _, err = diario.AbandonarCaptura(context.Background(), declaracionCopias(ses), reg.SolicitudAbandono{Operacion: s.Operacion, Clave: "abandonar:publicada", VersionEsperada: 1, SolicitudSHA256: s.SHA256, Destino: s.Destino, FalloReferencia: "verificacion_fallida", FalloSHA256: strings.Repeat("f", 64)}); err != nil {
		t.Fatal(err)
	}
	if _, err = lector.Detalle(context.Background(), ses, s.Conjunto); !errors.Is(err, p.ErrNoDisponible) || destino.llamadas != 1 {
		t.Fatal("abandono postpublicado oculta corrupcion", err)
	}
}

func TestListadoDosFilasAuditaCadaConjuntoYFalloBloqueaPagina(t *testing.T) {
	ses := sesionPrueba(t)
	base := t.TempDir()
	dir := filepath.Join(base, "diario")
	root := filepath.Join(base, "estado")
	if os.Mkdir(dir, 0700) != nil || os.Mkdir(root, 0700) != nil {
		t.Fatal("dirs")
	}
	diario, err := regadapter.Abrir(regadapter.Config{Directorio: dir, RaicesRestauradas: []string{root}, LimiteListado: 100})
	if err != nil {
		t.Fatal(err)
	}
	permisos := map[string]bool{}
	for _, numero := range []string{"uno", "dos"} {
		s := operacionescopias.Solicitud{Operacion: "operacion:" + numero, Clave: "clave:" + numero, SHA256: strings.Repeat("a", 64), Conjunto: "conjunto:" + numero, Destino: "destino:" + numero, Politica: "politica:fixture"}
		permisos[s.Operacion] = true
		permisos[s.Conjunto] = true
		if _, err = diario.Reservar(context.Background(), declaracionCopias(ses), s); err != nil {
			t.Fatal(err)
		}
	}
	auditor := &auditorCopiasPrueba{}
	destino := &destinoLecturasSintetico{err: p.ErrNoEncontrado}
	lector := &LecturasCopias{Diario: diario, Destino: destino, Autoridad: &autoridadCopiasPrueba{recursos: permisos}, Auditor: auditor}
	pagina, err := lector.Listar(context.Background(), ses, "", 25)
	if err != nil || len(pagina.Copias) != 2 || destino.llamadas != 0 {
		t.Fatal(pagina, err)
	}
	auditadas := map[string]int{}
	for _, recurso := range auditor.recursos {
		auditadas[recurso]++
	}
	if auditadas["conjunto:uno"] != 1 || auditadas["conjunto:dos"] != 1 || auditadas["copias"] != 1 {
		t.Fatal("no audita recursos exactos", auditadas)
	}
	auditor.falloRecurso = "conjunto:dos"
	if pagina, err = lector.Listar(context.Background(), ses, "", 25); !errors.Is(err, p.ErrNoDisponible) || len(pagina.Copias) != 0 {
		t.Fatal("auditabilidad parcial devuelve datos", pagina, err)
	}
}
