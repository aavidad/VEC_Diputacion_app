package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/cronos/domain"
	"vec-diputacion-granada/internal/modules/cronos/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type fuenteSolicitudOriginalC8Prueba struct {
	dato     ports.SolicitudOriginalC8
	llamadas int
}

func TestMaterialOriginalC8LigaDecisionAD273AlDocumento(t *testing.T) {
	m := domain.MaterialJustificacionOriginalC8{
		ActorRef: "per_" + strings.Repeat("A", 22), PerfilRef: "prf_" + strings.Repeat("B", 22),
		EmpleadoRef: "emp_" + strings.Repeat("C", 22), EmpleadoObjetivoRef: "emp_" + strings.Repeat("C", 22),
		SolicitudRef: "permiso:cronos:solicitud:abcdefgh", ClaveOperacion: "ref:" + strings.Repeat("d", 64),
		VersionEsperada: "0", DocumentoID: "ref:" + strings.Repeat("e", 64), DocumentoVersion: "1",
		DocumentoSHA256: strings.Repeat("f", 64), ReciboDocumentoRef: "recibo:documentos:abc",
		CustodioID: "almacen.local", PersonaTitularRef: "per_" + strings.Repeat("A", 22),
		PersonaAportanteRef: "per_" + strings.Repeat("A", 22), ContextoRef: "rca_" + strings.Repeat("G", 22),
		Accion: domain.AccionAnexarOriginalC8, ZonaHoraria: "Europe/Madrid",
	}
	canon, err := m.Canonico()
	if err != nil {
		t.Fatal(err)
	}
	var campos map[string]string
	if json.Unmarshal(canon, &campos) != nil || len(campos) != 19 {
		t.Fatalf("material no canónico: %s", canon)
	}
	r, err := RecursoJustificacionOriginalC8(m)
	if err != nil {
		t.Fatal(err)
	}
	huellaMaterial := sha256.Sum256(canon)
	esperada := `{"ambitos":{"empleado_ref":"` + m.EmpleadoObjetivoRef + `","solicitud_ref":"` + m.SolicitudRef +
		`"},"atributos":{"material_sha256":"` + hex.EncodeToString(huellaMaterial[:]) + `"}}`
	huellaEsperada := sha256.Sum256([]byte(esperada))
	huella, err := r.HuellaContextoAutorizacionSHA256()
	if err != nil || huella != hex.EncodeToString(huellaEsperada[:]) {
		t.Fatalf("huella V3 divergente: %s, %v", huella, err)
	}
	m.DocumentoID = "ref:" + strings.Repeat("1", 64)
	alterado, err := RecursoJustificacionOriginalC8(m)
	if err != nil {
		t.Fatal(err)
	}
	huellaNueva, _ := alterado.HuellaContextoAutorizacionSHA256()
	if huellaNueva == huella {
		t.Fatal("cambiar original conservó la misma autorización")
	}
}

func (f *fuenteSolicitudOriginalC8Prueba) PrepararOriginal(_ context.Context, _ ports.OrdenOriginalC8, _ string, _ ports.ModoLecturaOriginalC8) (ports.SolicitudOriginalC8, error) {
	f.llamadas++
	return f.dato, nil
}

type fuenteDocumentalOriginalC8Prueba struct {
	dato           domain.OriginalJustificacionC8V2
	llamadas       int
	llamadasPropio int
	llamadasRRHH   int
}

func (f *fuenteDocumentalOriginalC8Prueba) AcreditarOriginalPropio(_ context.Context, _ ports.OrdenOriginalC8, _ ports.SolicitudOriginalC8, _ string, _ uint64) (domain.OriginalJustificacionC8V2, error) {
	f.llamadas++
	f.llamadasPropio++
	return f.dato, nil
}

func (f *fuenteDocumentalOriginalC8Prueba) AcreditarOriginalRRHH(_ context.Context, _ ports.OrdenOriginalC8, _ ports.SolicitudOriginalC8, _ string, _ uint64) (domain.OriginalJustificacionC8V2, error) {
	f.llamadas++
	f.llamadasRRHH++
	return f.dato, nil
}

func TestOriginalC8V2AporteUsaSoloAutoridadDocumentalPropia(t *testing.T) {
	legacy, ordenVieja, _, _, _, _, peticion := escenarioJustificacion(t)
	actor, err := ordenVieja.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorOriginalC8Prueba{}
	orden, err := ports.NuevaOrdenOriginalC8(actor, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	empleados, err := actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		t.Fatal(err)
	}
	fuente := &fuenteSolicitudOriginalC8Prueba{dato: ports.SolicitudOriginalC8{
		SolicitudRef: peticion.SolicitudRef, EmpleadoRef: empleados[0], PersonaTitularRef: actor.PersonaRef,
		EstadoPermiso: domain.EstadoPermisoConcedido, VersionPermiso: 3, PendienteJustificar: true,
		JustificanteExigido: true, ContextoRef: "rca_" + strings.Repeat("C", 22)}}
	original := domain.OriginalJustificacionC8V2{DocumentoID: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 1,
		HuellaSHA256Bytes: strings.Repeat("b", 64), ReciboDocumentoRef: "recibo:documentos:abc",
		RecursoRef: peticion.SolicitudRef, PersonaTitularRef: actor.PersonaRef, PersonaAportanteRef: actor.PersonaRef,
		CustodioID: "almacen.local", ModuloID: "cronos", TipoRecurso: "justificacion",
		Finalidad: "custodiar_justificacion_cronos", EstadoAdmision: "admitido"}
	archivo := &fuenteDocumentalOriginalC8Prueba{dato: original}
	s, err := NuevoServicioJustificacionOriginalC8(fuente, archivo, &repositorioOriginalC8Prueba{},
		legacy.reloj, time.FixedZone("Europe/Madrid", 3600))
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Aportar(context.Background(), orden, ports.PeticionAnexoOriginalC8{
		SolicitudRef: peticion.SolicitudRef, ClaveOperacion: "ref:" + strings.Repeat("d", 64),
		DocumentoID: original.DocumentoID, DocumentoVersion: 1, VersionEsperada: 0})
	if archivo.llamadasPropio != 1 || archivo.llamadasRRHH != 0 || proveedor.llamadas != 1 {
		t.Fatalf("se usó autoridad ajena en aporte: propio=%d rrhh=%d proveedor=%d",
			archivo.llamadasPropio, archivo.llamadasRRHH, proveedor.llamadas)
	}
}

type proveedorOriginalC8Prueba struct{ llamadas int }

func (p *proveedorOriginalC8Prueba) ProveerMaterialOriginal(_ context.Context, _ domain.MaterialJustificacionOriginalC8) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	p.llamadas++
	return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, ports.ErrJustificacionNoDisponible
}

type repositorioOriginalC8Prueba struct{ llamadas int }

func (r *repositorioOriginalC8Prueba) ConfirmarOriginal(_ context.Context, _ ports.OrdenOriginalC8, _ domain.MaterialJustificacionOriginalC8,
	_ vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ReciboOriginalC8, error) {
	r.llamadas++
	return ports.ReciboOriginalC8{}, ports.ErrJustificacionNoDisponible
}

func TestOriginalC8V2NoAutorizaOtroTitularNiOtroRecurso(t *testing.T) {
	legacy, ordenVieja, _, _, _, _, peticion := escenarioJustificacion(t)
	actor, err := ordenVieja.ContextoActor()
	if err != nil {
		t.Fatal(err)
	}
	proveedor := &proveedorOriginalC8Prueba{}
	orden, err := ports.NuevaOrdenOriginalC8(actor, proveedor)
	if err != nil {
		t.Fatal(err)
	}
	empleados, err := actor.Referencias("empleado")
	if err != nil || len(empleados) != 1 {
		t.Fatalf("empleado del actor: %v", err)
	}
	ref := peticion.SolicitudRef
	base := ports.SolicitudOriginalC8{SolicitudRef: ref, EmpleadoRef: empleados[0], PersonaTitularRef: actor.PersonaRef,
		EstadoPermiso: domain.EstadoPermisoConcedido, VersionPermiso: 3, PendienteJustificar: true,
		JustificanteExigido: true, ContextoRef: "rca_" + strings.Repeat("C", 22)}
	original := domain.OriginalJustificacionC8V2{DocumentoID: "ref:" + strings.Repeat("a", 64), DocumentoVersion: 1,
		HuellaSHA256Bytes: strings.Repeat("b", 64), ReciboDocumentoRef: "recibo:documentos:abc", RecursoRef: ref,
		PersonaTitularRef: actor.PersonaRef, PersonaAportanteRef: actor.PersonaRef, CustodioID: "almacen.local",
		ModuloID: "cronos", TipoRecurso: "justificacion", Finalidad: "custodiar_justificacion_cronos", EstadoAdmision: "admitido"}
	for _, caso := range []string{"titular", "recurso", "huella"} {
		fuente := &fuenteSolicitudOriginalC8Prueba{dato: base}
		archivo := &fuenteDocumentalOriginalC8Prueba{dato: original}
		repo := &repositorioOriginalC8Prueba{}
		s, err := NuevoServicioJustificacionOriginalC8(fuente, archivo, repo, legacy.reloj, time.FixedZone("Europe/Madrid", 3600))
		if err != nil {
			t.Fatal(err)
		}
		switch caso {
		case "titular":
			fuente.dato.PersonaTitularRef = "per_" + strings.Repeat("Z", 22)
		case "recurso":
			archivo.dato.RecursoRef = "permiso:cronos:solicitud:otra1234"
		case "huella":
			archivo.dato.HuellaSHA256Bytes = ""
		}
		_, err = s.Aportar(context.Background(), orden, ports.PeticionAnexoOriginalC8{
			SolicitudRef: ref, ClaveOperacion: "ref:" + strings.Repeat("d", 64), DocumentoID: original.DocumentoID,
			DocumentoVersion: 1, VersionEsperada: 0})
		if err == nil || proveedor.llamadas != 0 || repo.llamadas != 0 {
			t.Fatalf("%s: efecto sin original propio: %v, proveedor=%d, repo=%d", caso, err, proveedor.llamadas, repo.llamadas)
		}
	}
}
