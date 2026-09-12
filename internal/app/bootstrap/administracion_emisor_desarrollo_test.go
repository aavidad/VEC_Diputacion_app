package bootstrap

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/config"
	admin "vec-diputacion-granada/internal/modules/administracion"
	adminhttp "vec-diputacion-granada/internal/modules/administracion/adapters/http"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	core "vec-diputacion-granada/internal/vec/domain"
	vp "vec-diputacion-granada/internal/vec/ports"
)

type relojEmisorAdministracionPrueba struct{ ahora time.Time }

func (r relojEmisorAdministracionPrueba) Ahora() time.Time { return r.ahora }

func materialEmisorAdministracionPrueba(t *testing.T) (archivoEmisorAdministracionDesarrollo, []byte, []byte, relojEmisorAdministracionPrueba) {
	t.Helper()
	reloj := relojEmisorAdministracionPrueba{time.Date(2026, 9, 12, 12, 0, 0, 0, time.UTC)}
	semilla, secreto := bytes.Repeat([]byte{7}, 32), bytes.Repeat([]byte{9}, 32)
	c := archivoEmisorAdministracionDesarrollo{Version: 1, Autoridad: AutoridadNoAutoritativa, CuentaRef: "cta_aaaaaaaaaaaaaaaaaaaaaa", CuentaOrdinariaRef: "cta_bbbbbbbbbbbbbbbbbbbbbb", PerfilRef: "prf_cccccccccccccccccccccc", PersonaRef: "per_dddddddddddddddddddddd", FuenteDSNFile: "fuente.dsn", MotivosDSNFile: "motivos.dsn"}
	c.Motivo = core.ReferenciaEntradaCatalogo{CatalogoID: catalogoMotivosAdministracionDesarrollo, CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_" + strings.Repeat("a", 32)}
	c.Firma.File, c.Firma.ClaveID, c.Firma.Version = "firma.seed", "clave:atestacion:administracion:desarrollo:v1", 1
	c.Firma.Desde, c.Firma.Hasta = reloj.ahora.Add(-time.Hour), reloj.ahora.Add(time.Hour)
	publica := ed25519.NewKeyFromSeed(semilla).Public().(ed25519.PublicKey)
	spki, err := x509.MarshalPKIXPublicKey(publica)
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.Sum256(spki)
	c.Firma.SPKISHA256 = hex.EncodeToString(h[:])
	c.Confianza.Referencia, c.Confianza.Orden = "confianza:atestacion:administracion:desarrollo:2026-09-12", 1
	c.Confianza.PublicadaEn, c.Confianza.ExpiraEn = c.Firma.Desde, c.Firma.Hasta
	raiz, err := confianza.NuevaRaizPublicaAtestacionAutorizacionV3EdDSA(c.Firma.ClaveID, c.Firma.Version, publica, audienciaAtestacionAdministracionDesarrollo, confianza.EstadoClaveAtestacionAutorizacionV3Activa, c.Firma.Desde, c.Firma.Hasta, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	conf, err := confianza.NuevaConfiguracionConfianzaAtestacionAutorizacionV3(c.Confianza.Referencia, c.Confianza.Orden, c.Confianza.PublicadaEn, c.Confianza.ExpiraEn, raiz)
	if err != nil {
		t.Fatal(err)
	}
	c.Confianza.HuellaSHA256, err = conf.HuellaSHA256ParaGobierno()
	if err != nil {
		t.Fatal(err)
	}
	h = sha256.Sum256(secreto)
	c.Capacidad = archivoCapacidadIncorporacionV2{ClaveID: "clave:capacidad:administracion:desarrollo:v1", Version: 1, File: "capacidad.key", SHA256: hex.EncodeToString(h[:]), EmisorID: "emisor:administracion:desarrollo:v1", Desde: c.Firma.Desde, Hasta: c.Firma.Hasta, RevisionGobierno: 1, HuellaGobierno: strings.Repeat("b", 64)}
	return c, semilla, secreto, reloj
}

func TestMaterialEmisorAdministracionConstruyeCadenaRealPropia(t *testing.T) {
	c, semilla, secreto, reloj := materialEmisorAdministracionPrueba(t)
	a, confianzaReal, capacidades, firmante, err := materialEmisorAdministracionDesarrollo(c, semilla, secreto, reloj)
	if err != nil || a == nil || confianzaReal == nil || capacidades == nil || firmante == nil {
		t.Fatalf("cadena incompleta: %v", err)
	}
	defer firmante.cerrar()
	claveEsperada := ed25519.NewKeyFromSeed(append([]byte(nil), semilla...))
	borrarBytes(semilla)
	borrarBytes(secreto)
	if !bytes.Equal(firmante.privada, claveEsperada) || !strings.HasPrefix(firmante.claveID, "clave:atestacion:administracion:") {
		t.Fatal("firma dependiente del buffer del cargador o de una clave ajena")
	}
	for _, valor := range []any{firmante, *firmante} {
		for _, formato := range []string{"%v", "%+v", "%#v"} {
			if fmt.Sprintf(formato, valor) != "firmanteAdministracion{redactado}" {
				t.Fatal("firmante revela material privado")
			}
		}
		b, err := json.Marshal(valor)
		if err != nil || string(b) != `{"redactado":true}` {
			t.Fatal("JSON revela material privado")
		}
	}
	if _, err := firmante.FirmarAtestacionAutorizacionV3(context.Background(), vp.SolicitudFirmaAtestacionAutorizacionV3{}); err == nil {
		t.Fatal("firmante acepta solicitud vacía")
	}
	firmante.cerrar()
	if len(firmante.privada) != 0 {
		t.Fatal("el cierre conserva la clave de firma")
	}
}

func TestMaterialEmisorAdministracionDeniegaClavesCruzadasOCaducadas(t *testing.T) {
	for _, nombre := range []string{"firma_ct", "capacidad_ct", "emisor_ct", "confianza_ct", "spki_ajeno", "gobierno_ajeno", "hmac_ajeno", "firma_caducada", "capacidad_caducada", "confianza_caducada", "semilla_cero", "hmac_cero", "claves_iguales", "semilla_corta"} {
		t.Run(nombre, func(t *testing.T) {
			c, semilla, secreto, reloj := materialEmisorAdministracionPrueba(t)
			switch nombre {
			case "firma_ct":
				c.Firma.ClaveID = "clave:atestacion:ct:desarrollo:v1"
			case "capacidad_ct":
				c.Capacidad.ClaveID = "clave:capacidad:ct:desarrollo:v1"
			case "emisor_ct":
				c.Capacidad.EmisorID = "emisor:ct:desarrollo:v1"
			case "confianza_ct":
				c.Confianza.Referencia = "confianza:atestacion:ct:desarrollo:v1"
			case "spki_ajeno":
				c.Firma.SPKISHA256 = strings.Repeat("c", 64)
			case "gobierno_ajeno":
				c.Confianza.HuellaSHA256 = strings.Repeat("c", 64)
			case "hmac_ajeno":
				c.Capacidad.SHA256 = strings.Repeat("c", 64)
			case "firma_caducada":
				c.Firma.Hasta = reloj.ahora
			case "capacidad_caducada":
				c.Capacidad.Hasta = reloj.ahora
			case "confianza_caducada":
				c.Confianza.ExpiraEn = reloj.ahora
			case "semilla_cero":
				semilla = make([]byte, 32)
			case "hmac_cero":
				secreto = make([]byte, 32)
			case "claves_iguales":
				secreto = append([]byte(nil), semilla...)
			case "semilla_corta":
				semilla = semilla[:31]
			}
			a, v, e, f, err := materialEmisorAdministracionDesarrollo(c, semilla, secreto, reloj)
			if err == nil || a != nil || v != nil || e != nil || f != nil {
				t.Fatal("material incoherente produce una cadena de emisión")
			}
		})
	}
}

func TestEmisorAdministracionCargaConfiguracionPrivadaCerrada(t *testing.T) {
	c, _, _, _ := materialEmisorAdministracionPrueba(t)
	directorio := t.TempDir()
	base := filepath.Join(directorio, "administracion")
	if err := os.Mkdir(base, 0700); err != nil {
		t.Fatal(err)
	}
	contenido, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(base, "emisor-v3.json")
	escribir := func(b []byte) {
		t.Helper()
		if err := os.WriteFile(ruta, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	escribir(contenido)
	cargada, raiz, err := leerEmisorAdministracionDesarrollo(directorio)
	if err != nil || raiz == nil || cargada.Motivo != c.Motivo {
		t.Fatalf("configuración válida rechazada: %v", err)
	}
	raiz.Close()
	for _, nombre := range []string{"campo_ajeno", "clave_duplicada", "catalogo_ct", "ruta_fuera", "permisos_publicos"} {
		t.Run(nombre, func(t *testing.T) {
			escribir(contenido)
			switch nombre {
			case "campo_ajeno":
				escribir(append([]byte(`{"ajeno":1,`), contenido[1:]...))
			case "clave_duplicada":
				escribir(append([]byte(`{"version":1,`), contenido[1:]...))
			case "catalogo_ct":
				otro := c
				otro.Motivo.CatalogoID = "motivos_autorizacion"
				b, _ := json.Marshal(otro)
				escribir(b)
			case "ruta_fuera":
				otro := c
				otro.Firma.File = "../ct/clave.seed"
				b, _ := json.Marshal(otro)
				escribir(b)
			case "permisos_publicos":
				if err := os.Chmod(ruta, 0644); err != nil {
					t.Fatal(err)
				}
				defer os.Chmod(ruta, 0600)
			}
			_, raiz, err := leerEmisorAdministracionDesarrollo(directorio)
			if raiz != nil {
				raiz.Close()
			}
			if err == nil {
				t.Fatal("cargador acepta configuración fuera del contrato")
			}
		})
	}
}

func TestEmisorAdministracionAusenteCierraYParcialNoCompone(t *testing.T) {
	ctx := context.Background()
	if d, err := nuevasDependenciasEmisorAdministracionDesarrollo(ctx, config.Config{}, nil, nil, nil); d != nil || err != nil {
		t.Fatal("ausencia opcional no cerrada")
	}
	if d, err := nuevasDependenciasEmisorAdministracionDesarrollo(ctx, config.Config{}, &identidadAdministracionDesarrollo{}, nil, nil); d != nil || err == nil {
		t.Fatal("identidad parcial compone un emisor")
	}
	if d, err := nuevasDependenciasEmisorAdministracionDesarrollo(nil, config.Config{}, &identidadAdministracionDesarrollo{}, nil, nil); d != nil || err == nil {
		t.Fatal("contexto ausente compone un emisor")
	}
}

func TestEmisorAdministracionNoAutorizaSinSesionLigada(t *testing.T) {
	delegado := &autorizadorPermisoAdministracionPrueba{}
	a := &autoridadEmisorAdministracionDesarrollo{delegado: delegado, reloj: relojEmisorAdministracionPrueba{time.Now().UTC()}}
	_, _, err := a.ExigirSolicitudLigadaV3(context.Background(), core.SolicitudAutorizacionLigadaV3{}, core.ResultadoContextoActorRegistradoV2{})
	if err == nil || delegado.llamadasActuales() != 0 {
		t.Fatal("autoridad delega una solicitud sin sesión ADMIN")
	}
}

// Las autoridades de identidad son dobles explícitos. Se usan los constructores
// nominales reales; esta prueba no acredita registro ni concesiones PostgreSQL.
type identidadEmisorAdministracionPrueba struct {
	autenticacion core.AutenticacionRevalidadaV1
	resultado     core.ResultadoContextoActorRegistradoV2
}

func (i identidadEmisorAdministracionPrueba) RevalidarAutenticacionActorV1(context.Context, core.SolicitudRevalidacionAutenticacionActorV1) (core.AutenticacionRevalidadaV1, error) {
	return i.autenticacion, nil
}
func (i identidadEmisorAdministracionPrueba) ResolverContextoActorRegistradoV2(context.Context, core.SolicitudContextoActor) (core.ResultadoContextoActorRegistradoV2, error) {
	return i.resultado, nil
}

func solicitudEmisorAdministracionPrueba(t *testing.T, c archivoEmisorAdministracionDesarrollo, reloj relojEmisorAdministracionPrueba) (core.SolicitudAutorizacionLigadaV3, core.ResultadoContextoActorRegistradoV2, core.DecisionAutorizacionLigadaV3) {
	t.Helper()
	ahora := reloj.ahora
	cuenta := core.CuentaAutenticadaContextoActor{CuentaRef: c.CuentaRef, Metodo: core.AuthMethodCertificate, Garantia: core.AuthAssuranceHigh}
	inst := core.InstantaneaContextoActor{VinculoRef: "vca_0123456789abcdefghijkl", VinculoVersion: 1, CuentaRef: c.CuentaRef, CuentaVersion: 1, PersonaRef: c.PersonaRef, PersonaVersion: 1, PerfilActivoRef: c.PerfilRef, PerfilVersion: 1, Estado: core.EstadoVinculoContextoActorActivo, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour)}
	actor, err := core.NuevoContextoActor(cuenta, inst, ahora)
	if err != nil {
		t.Fatal(err)
	}
	acreditacion := core.AcreditacionProcedenciaComponenteContextoActorV1{ProcedenciaRef: "prc_0123456789abcdefghijkl", ProcedenciaVersion: 1, ProcedenciaHuellaSHA256: strings.Repeat("4", 64), ProcedenciaAutoridad: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1}
	manifiesto := core.ManifiestoProcedenciaContextoActorV1{
		Esquema: core.EsquemaManifiestoProcedenciaContextoActorV1, AutoridadEfectiva: core.AutoridadProcedenciaContextoActorMaestraAcreditadaV1,
		Cuenta:   core.ProcedenciaCuentaContextoActorV1{CuentaRef: c.CuentaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Persona:  core.ProcedenciaPersonaContextoActorV1{PersonaRef: c.PersonaRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Perfil:   core.ProcedenciaPerfilContextoActorV1{PerfilRef: c.PerfilRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Contexto: core.ProcedenciaVinculoContextoActorV1{VinculoRef: inst.VinculoRef, Version: 1, AcreditacionProcedenciaComponenteContextoActorV1: acreditacion},
		Vinculos: []core.ProcedenciaVinculoReferenciaContextoActorV1{},
	}
	r := core.ResultadoContextoActorRegistradoV2{RegistroContextoRef: "rca_0123456789abcdefghijklmn", Contexto: actor, AutoridadEfectiva: manifiesto.AutoridadEfectiva, ResueltoEnAutoritativo: actor.ResueltoEn}
	r.RepresentacionCanonica, err = actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	r.HuellaSHA256, err = actor.HuellaSHA256VinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	r.ManifiestoProcedenciaCanonico, err = manifiesto.RepresentacionCanonicaV1()
	if err != nil {
		t.Fatal(err)
	}
	r.ManifiestoProcedenciaHuellaSHA256, err = core.HuellaSHA256ManifiestoProcedenciaContextoActorV1(r.ManifiestoProcedenciaCanonico)
	if err != nil || r.Validar() != nil {
		t.Fatal("contexto nominal de prueba inválido", err)
	}
	aut := core.AutenticacionRevalidadaV1{AutenticacionRef: "aut_0123456789abcdefghijkl", AutenticacionHuellaSHA256: strings.Repeat("1", 64), AsercionRef: "ase_0123456789abcdefghijkl", SesionRef: "ses_0123456789abcdefghijkl", ControlSesionRef: "cse_0123456789abcdefghijkl", ControlSesionRevision: 1, ControlSesionHuellaSHA256: strings.Repeat("2", 64), CuentaRef: c.CuentaRef, CuentaOrdinariaRef: c.CuentaOrdinariaRef, CuentaPrivilegiada: true, Superficie: core.SuperficieAutenticacionAdministracionPrivilegiadaV1, MetodoObservado: cuenta.Metodo, GarantiaObservada: cuenta.Garantia, PoliticaGarantiaRef: "pga_0123456789abcdefghijkl", PoliticaGarantiaHuellaSHA256: strings.Repeat("3", 64), AutenticacionVerificadaEn: ahora.Add(-3 * time.Minute), SesionEmitidaEn: ahora.Add(-2 * time.Minute), SesionRevalidadaEn: ahora, SesionValidaHasta: ahora.Add(time.Minute)}
	i := identidadEmisorAdministracionPrueba{aut, r}
	v, err := core.CrearVinculoAutenticacionActorV2(context.Background(), i, core.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: aut.AutenticacionRef, SesionRef: aut.SesionRef}, i, core.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: c.PerfilRef}, reloj)
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := core.GenerarReferenciaCorrelacionAutorizacionV2(context.Background(), &referenciasPermisoAdministracionPrueba{})
	if err != nil {
		t.Fatal(err)
	}
	s, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{VinculoAutenticacionActor: v, ReferenciaMotivo: c.Motivo, Accion: accionConfiguracionCorreoAdministracionV3, Recurso: core.RecursoAutorizable{Referencia: referenciaConfiguracionCorreoAdministracionV3, ModuloID: admin.ModuleID, Tipo: tipoRecursoConfiguracionCorreoAdministracion, Ambitos: map[string]string{"organizacion_ref": organizacionConfiguracionCorreoAdministracionV3}, Atributos: map[string]string{"material_sha256": strings.Repeat("e", 64)}}, Finalidad: finalidadConfiguracionCorreoAdministracionV3, Correlacion: correlacion})
	if err != nil {
		t.Fatal(err)
	}
	rol := core.VersionRol{RolID: "administrador", Version: 1, Nombre: "Administración sintética", Estado: core.EstadoVersionRolPublicada, Concesiones: []core.ConcesionRol{{Accion: accionConfiguracionCorreoAdministracionV3, ModuloID: admin.ModuleID, TipoRecurso: tipoRecursoConfiguracionCorreoAdministracion, Finalidades: []string{finalidadConfiguracionCorreoAdministracionV3}, GarantiaMinima: core.AuthAssuranceHigh}}, PublicadaPor: "responsable-seguridad", PublicadaEn: ahora.Add(-time.Hour)}
	huella, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
	if err != nil {
		t.Fatal(err)
	}
	instantanea := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-admin", Version: 1, PerfilActivoRef: c.PerfilRef, PrincipalID: c.PersonaRef, VersionRolRef: rol.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, Ambitos: []core.AmbitoPerfil{{Clave: "organizacion_ref", Valores: []string{organizacionConfiguracionCorreoAdministracionV3}}}, VigenteDesde: ahora.Add(-time.Hour), VigenteHasta: ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: ahora.Add(-time.Hour)}, VersionRol: rol, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: rol.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: rol.PublicadaPor, ActualizadoEn: rol.PublicadaEn}, RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huella}
	evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(s, instantanea, "dec_0123456789abcdef0123456789abcdef", ahora, ahora.Add(30*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decision, err := core.NuevaDecisionAutorizacionLigadaV3(s, evidencia)
	if err != nil {
		t.Fatal(err)
	}
	concedida, _, err := decision.Resultado()
	if err != nil || !concedida {
		t.Fatal("decisión de prueba no concedida", err)
	}
	return s, r, decision
}

func TestMaterialEmisorAdministracionFirmaVerificaYEmiteCapacidadPropia(t *testing.T) {
	c, semilla, secreto, reloj := materialEmisorAdministracionPrueba(t)
	solicitud, resultado, decision := solicitudEmisorAdministracionPrueba(t, c, reloj)
	a, verificador, capacidades, firmante, err := materialEmisorAdministracionDesarrollo(c, semilla, secreto, reloj)
	if err != nil {
		t.Fatal(err)
	}
	defer firmante.cerrar()
	atestacion, err := a.Atestar(context.Background(), decision, c.Motivo, resultado)
	if err != nil {
		t.Fatal("firma ADMIN válida", err)
	}
	prueba, err := verificador.Verificar(context.Background(), solicitud, decision, c.Motivo, resultado, atestacion)
	if err != nil {
		t.Fatal("verificación ADMIN válida", err)
	}
	if _, err := capacidades.Emitir(context.Background(), solicitud, decision, c.Motivo, resultado, atestacion, prueba); err != nil {
		t.Fatal("capacidad ADMIN válida", err)
	}
	firmante.cerrar()
	if _, err := a.Atestar(context.Background(), decision, c.Motivo, resultado); err == nil {
		t.Fatal("el firmante cerrado conserva capacidad de emisión")
	}
}

func TestEmisorAdministracionAutorizaPersonaConSujetoCertificadoDistinto(t *testing.T) {
	c, _, _, reloj := materialEmisorAdministracionPrueba(t)
	solicitud, resultado, _ := solicitudEmisorAdministracionPrueba(t, c, reloj)
	m := nuevoMaterialTLSAdministracionPrueba(t)
	transporte, _, cfg := nuevaAutoridadAdministracionPrueba(t, m, true)
	transporte.identidad.identidad.principal.ID = "administracion-sintetica"
	transporte.identidad.cuentaRef, transporte.identidad.cuentaOrdinariaRef = c.CuentaRef, c.CuentaOrdinariaRef
	transporte.identidad.personaRef, transporte.identidad.perfilRef = c.PersonaRef, c.PerfilRef
	delegado := &autorizadorPermisoAdministracionPrueba{}
	a := &autoridadEmisorAdministracionDesarrollo{delegado: delegado, identidad: *transporte.identidad, motivo: c.Motivo, reloj: reloj, soloEscritura: true}
	observacion := make(chan error, 1)
	url := iniciarServidorAdministracionPrueba(t, transporte, cfg, m, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _, err := a.ExigirSolicitudLigadaV3(r.Context(), solicitud, resultado)
		if err != ErrConfiguracionCorreoAdministracionNoDisponible || delegado.llamadasActuales() != 1 {
			observacion <- fmt.Errorf("persona ligada no alcanzó PDP: llamadas=%d err=%v", delegado.llamadasActuales(), err)
		} else {
			var fallo error
			for _, caso := range []string{"organizacion_ajena", "ambito_ausente", "huella_mayuscula", "acceso_sin_escritura"} {
				d, e := solicitud.Datos()
				if e != nil {
					fallo = e
					break
				}
				switch caso {
				case "organizacion_ajena":
					d.Recurso.Ambitos["organizacion_ref"] = "organizacion:otra"
				case "ambito_ausente":
					d.Recurso.Ambitos = nil
				case "huella_mayuscula":
					d.Recurso.Atributos["material_sha256"] = strings.Repeat("E", 64)
				case "acceso_sin_escritura":
					d.Accion = admin.PermissionIntegrationsManage
					d.Recurso.Atributos = nil
				}
				cruzada, e := core.NuevaSolicitudAutorizacionLigadaV3(d)
				if e != nil {
					fallo = fmt.Errorf("fixture negativa %s: %w", caso, e)
					break
				}
				_, _, e = a.ExigirSolicitudLigadaV3(r.Context(), cruzada, resultado)
				if e == nil || delegado.llamadasActuales() != 1 {
					fallo = fmt.Errorf("solicitud %s alcanzó PDP", caso)
					break
				}
			}
			observacion <- fallo
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	cliente := clienteAdministracionPrueba(m, &m.admin)
	defer cliente.CloseIdleConnections()
	peticion, err := http.NewRequest(http.MethodPut, url+adminhttp.RutaConfiguracionCorreo, nil)
	if err != nil {
		t.Fatal(err)
	}
	respuesta, err := cliente.Do(peticion)
	if err != nil {
		t.Fatal(err)
	}
	respuesta.Body.Close()
	if respuesta.StatusCode != http.StatusNoContent {
		t.Fatalf("estado TLS %d", respuesta.StatusCode)
	}
	if err := <-observacion; err != nil {
		t.Fatal(err)
	}
}
