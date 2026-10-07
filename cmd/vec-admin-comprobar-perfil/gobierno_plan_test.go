package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/domain"
)

func archivosPlanGobiernoPrueba(t *testing.T, operacion domain.OperacionGobiernoPerfil) (string, string, string) {
	t.Helper()
	rc, rp, fecha := archivosPrueba(t)
	var c domain.CatalogoAccionesAdministracionV1
	var p domain.PropuestaPerfilAdministracionV1
	for ruta, destino := range map[string]any{rc: &c, rp: &p} {
		datos, err := os.ReadFile(ruta)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(datos, destino); err != nil {
			t.Fatal(err)
		}
	}
	s := domain.SolicitudPlanGobiernoPerfil{Operacion: operacion, Publicacion: &p,
		Motivo: domain.ReferenciaEntradaCatalogo{CatalogoID: "motivos_administracion", CatalogoVersion: 1,
			CatalogoHuellaSHA256: strings.Repeat("c", 64), EntradaClave: "motivo_" + strings.Repeat("d", 32)}}
	if operacion != domain.OperacionCrearPerfilGobernado {
		base := p.RolPropuesto
		control := domain.ControlVigenciaVersionRol{VersionRolRef: base.Referencia(), Revision: 1, Estado: domain.EstadoControlVigenciaVersionRolHabilitada,
			ActualizadoPor: base.PublicadaPor, ActualizadoEn: base.PublicadaEn}
		c.Perfiles = []domain.PerfilPublicadoAdministracionV1{{Rol: base, ControlVigencia: control, TipoPerfil: domain.TipoPerfilAdministracionAdministrableV1}}
		hc, err := c.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		hr, err := base.HuellaSHA256()
		if err != nil {
			t.Fatal(err)
		}
		if operacion == domain.OperacionVersionarPerfilGobernado {
			p.CatalogoHuellaSHA256, p.VersionRolBaseRef, p.VersionRolBaseHuellaSHA256 = hc, base.Referencia(), hr
			p.RolPropuesto.Version = 2
		} else {
			hcontrol, err := control.HuellaSHA256()
			if err != nil {
				t.Fatal(err)
			}
			s.Publicacion = nil
			s.Deshabilitacion = &domain.SeleccionRetiradaVersionPerfil{CatalogoRef: c.Referencia, CatalogoVersion: c.Version, CatalogoHuellaSHA256: hc,
				VersionRolRef: base.Referencia(), VersionRolHuellaSHA256: hr, ControlRevision: control.Revision, ControlHuellaSHA256: hcontrol}
		}
	}
	for ruta, valor := range map[string]any{rc: c, rp: s} {
		datos, err := json.Marshal(valor)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ruta, datos, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return rc, rp, fecha
}

func TestCLIPlanGobiernoCrearVersionarDeshabilitarESEN(t *testing.T) {
	for _, idioma := range []string{"es", "en"} {
		for _, operacion := range []domain.OperacionGobiernoPerfil{domain.OperacionCrearPerfilGobernado, domain.OperacionVersionarPerfilGobernado, domain.OperacionDeshabilitarVersionPerfil} {
			t.Run(idioma+"/"+string(operacion), func(t *testing.T) {
				c, s, fecha := archivosPlanGobiernoPrueba(t, operacion)
				textos := filepath.Join("../../web/static/textos", idioma, "admin-comprobar-perfil.json")
				var salida bytes.Buffer
				args := []string{textos, c, s, fecha, "--preparar-plan"}
				if codigo := ejecutar(args, &salida); codigo != 0 {
					t.Fatalf("exit=%d salida=%s", codigo, salida.String())
				}
				var r salidaComprobador
				if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
					t.Fatal(err)
				}
				if !r.Comprobado || r.Publicado || r.Dictamen != nil || r.Plan == nil || r.Mensaje == r.Codigo || r.Limite == "" || r.Plan.Operacion != operacion {
					t.Fatalf("resultado=%+v", r)
				}
				h, err := r.Plan.HuellaSHA256()
				if err != nil || h != r.PlanHuellaSHA256 {
					t.Fatal("plan sin huella reproducible")
				}
				salida.Reset()
				if codigo := ejecutar(args, &salida); codigo != 0 {
					t.Fatalf("replay exit=%d", codigo)
				}
				var replay salidaComprobador
				if err := json.Unmarshal(salida.Bytes(), &replay); err != nil {
					t.Fatal(err)
				}
				if replay.PlanHuellaSHA256 != h || replay.Publicado {
					t.Fatal("otra preparación altera la huella o publica")
				}
			})
		}
	}
}

func TestCLIPlanGobiernoNoAceptaIdentidadNiPermisoDeArchivos(t *testing.T) {
	c, s, fecha := archivosPlanGobiernoPrueba(t, domain.OperacionCrearPerfilGobernado)
	textos := filepath.Join("../../web/static/textos/es", "admin-comprobar-perfil.json")
	datos, err := os.ReadFile(s)
	if err != nil {
		t.Fatal(err)
	}
	datos = bytes.Replace(datos, []byte(`{"operacion":`), []byte(`{"actor":{"administrador":true},"operacion":`), 1)
	if err := os.WriteFile(s, datos, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	if codigo := ejecutar([]string{textos, c, s, fecha, "--preparar-plan"}, &salida); codigo != 1 {
		t.Fatalf("exit=%d", codigo)
	}
	var r salidaComprobador
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Plan != nil || r.Publicado || r.PlanHuellaSHA256 != "" || r.Codigo != errEntrada.Error() {
		t.Fatal("atributos de identidad del archivo produjeron plan")
	}
}

func TestCLIPlanGobiernoRetiradaRechazaControlCruzado(t *testing.T) {
	c, s, fecha := archivosPlanGobiernoPrueba(t, domain.OperacionDeshabilitarVersionPerfil)
	var intencion domain.SolicitudPlanGobiernoPerfil
	if err := leerJSON(s, &intencion, 4<<20); err != nil {
		t.Fatal(err)
	}
	intencion.Deshabilitacion.ControlRevision++
	datos, err := json.Marshal(intencion)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(s, datos, 0600); err != nil {
		t.Fatal(err)
	}
	var salida bytes.Buffer
	textos := filepath.Join("../../web/static/textos/es", "admin-comprobar-perfil.json")
	if codigo := ejecutar([]string{textos, c, s, fecha, "--preparar-plan"}, &salida); codigo != 1 {
		t.Fatalf("exit=%d", codigo)
	}
	var r salidaComprobador
	if err := json.Unmarshal(salida.Bytes(), &r); err != nil {
		t.Fatal(err)
	}
	if r.Plan != nil || r.Publicado || r.Codigo != domain.ErrOrigenPerfilAdministracionNoCoincide.Error() {
		t.Fatal("control ajeno produce plan de retirada")
	}
}
