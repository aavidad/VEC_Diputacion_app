package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	bolsahttp "vec-diputacion-granada/internal/modules/bolsa/adapters/httpinterno"
	bolsadomain "vec-diputacion-granada/internal/modules/bolsa/domain"
	bolsaports "vec-diputacion-granada/internal/modules/bolsa/ports"
	ctports "vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
	seguridadvec "vec-diputacion-granada/internal/vec/adapters/seguridad"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

func TestFuentePoliticaCeseCoherenteConConcesion(t *testing.T) {
	contenido, err := os.ReadFile("../../../deploy/principal/fuentes/bolsa_politica_cese_rrhh_v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var fuente struct {
		Referencia string                                    `json:"referencia"`
		Version    int                                       `json:"version"`
		Huella     string                                    `json:"huella_sha256"`
		Canon      string                                    `json:"entradas_canon"`
		Entradas   []vecdomain.EntradaAccionAdministracionV1 `json:"entradas"`
	}
	if err := json.Unmarshal(contenido, &fuente); err != nil {
		t.Fatal(err)
	}
	huella := sha256.Sum256([]byte(fuente.Canon))
	if hex.EncodeToString(huella[:]) != fuente.Huella || len(fuente.Entradas) != 1 || strings.Contains(fuente.Canon, "fuente_huella_sha256") {
		t.Fatal("huella/canon de la fuente incorrectos")
	}
	e := fuente.Entradas[0]
	if e.Validar() != nil || e.FuenteRef != fuente.Referencia || e.FuenteVersion != fuente.Version || e.FuenteHuellaSHA256 != fuente.Huella ||
		e.Concesion.Accion != bolsaports.AccionConsultarPoliticaCese || e.Concesion.TipoRecurso != bolsaports.TipoRecursoPoliticaCese ||
		len(e.Concesion.Finalidades) != 1 || e.Concesion.Finalidades[0] != bolsaports.FinalidadConsultarPoliticaCese ||
		len(e.Concesion.CamposPermitidos) != 1 || e.Concesion.CamposPermitidos[0] != bolsaports.CampoPoliticaCese {
		t.Fatal("la entrada no pertenece a su fuente o no coincide con el lector")
	}
}

type registroPoliticaCesePrueba struct {
	registradorPerfilBolsaPrueba
	fallar bool
}

func (r *registroPoliticaCesePrueba) AppendIntentoAuditoria(ctx context.Context, orden vecports.OrdenIntentoAuditoria) (vecports.AcuseIntentoAuditoria, error) {
	if r.fallar {
		r.llamadas++
		return vecports.AcuseIntentoAuditoria{}, vecports.ErrIntentoAuditoriaNoDisponible
	}
	return r.registradorPerfilBolsaPrueba.AppendIntentoAuditoria(ctx, orden)
}

func TestRutaPoliticaCeseMontadaConPermisoActualYConstancia(t *testing.T) {
	for _, caso := range []struct {
		nombre, cambio   string
		codigo, lecturas int
	}{
		{"publicada_v2", "", http.StatusOK, 1},
		{"sin_permiso", "sin_permiso", http.StatusForbidden, 0},
		{"rol_retirado", "rol_retirado", http.StatusForbidden, 0},
		{"campo_ajeno", "campo_ajeno", http.StatusForbidden, 0},
		{"ambito_ajeno", "ambito_ajeno", http.StatusForbidden, 0},
		{"fuente_ausente", "fuente_ausente", http.StatusServiceUnavailable, 0},
		{"lectura_fallida", "lectura_fallida", http.StatusServiceUnavailable, 1},
		{"auditoria_fallida", "auditoria_fallida", http.StatusServiceUnavailable, 1},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			ahora := time.Now().UTC().Truncate(time.Microsecond)
			actor, fuente := escenarioLecturaRespuestasLlamamiento(t, ahora)
			fuente.i.VersionRol.Concesiones = append(fuente.i.VersionRol.Concesiones, vecdomain.ConcesionRol{
				Accion: bolsaports.AccionConsultarPoliticaCese, ModuloID: "bolsa", TipoRecurso: bolsaports.TipoRecursoPoliticaCese,
				Finalidades: []string{bolsaports.FinalidadConsultarPoliticaCese}, GarantiaMinima: vecdomain.AuthAssuranceHigh,
				CamposPermitidos: []string{bolsaports.CampoPoliticaCese}})
			fronteras, err := descriptoresFronterasBorradorLlamamientoBolsaDesarrollo(actor.Resultado.Contexto.PerfilActivoRef)
			if err != nil {
				t.Fatal(err)
			}
			catalogo, err := nuevoCatalogoFronterasComunDesarrollo(fronteras)
			if err != nil {
				t.Fatal(err)
			}
			descriptor, ok := catalogo.resolver(http.MethodGet, bolsahttp.RutaPoliticaCese)
			if !ok {
				t.Fatal("falta la frontera GET de política de cese")
			}
			if _, ok := catalogo.resolver(http.MethodHead, bolsahttp.RutaPoliticaCese); !ok {
				t.Fatal("falta HEAD")
			}
			if _, ok := catalogo.resolver(http.MethodPost, bolsahttp.RutaPoliticaCese); ok {
				t.Fatal("la lectura admite escritura")
			}
			ctx := context.WithValue(context.Background(), claveFronteraSeguridadComunDesarrollo{}, fronteraSeguridadComunDesarrollo{
				metodo: http.MethodGet, ruta: bolsahttp.RutaPoliticaCese, superficie: superficieInternaSeguridadComunDesarrollo,
				catalogo: catalogo, descriptor: descriptor})
			seguridad, err := nuevaSeguridadComunDesarrollo(proveedorContextoComunConsultasRRHHDesarrolloPrueba{
				resolver: func(context.Context) (contextoSeguridadComunDesarrollo, error) { return actor, nil }}, &relojSesionConsultaPrueba{ahora: ahora})
			if err != nil {
				t.Fatal(err)
			}
			p := &preparadorBorradorLlamamientoDesarrollo{sesion: seguridad, fuenteRespuestas: fuente,
				generar: seguridadvec.GeneradorReferenciasCriptograficas{}, soporte: &soporteSesionBorradorBolsaDesarrollo{
					unidadRef: "unidad:prueba", ambitoRef: "ambito:prueba", soporteCanal: &soporteAltaContratacionTemporalDesarrollo{
						contexto: ctports.ContextoAutorizacionAltaV3{Vinculo: actor.Vinculo, Resultado: actor.Resultado}}}}
			registro := &registroPoliticaCesePrueba{}
			switch caso.cambio {
			case "sin_permiso":
				fuente.i.VersionRol.Concesiones = fuente.i.VersionRol.Concesiones[:len(fuente.i.VersionRol.Concesiones)-1]
			case "rol_retirado":
				fuente.i.ControlVigenciaVersionRol.Estado = vecdomain.EstadoControlVigenciaVersionRolRetirada
				fuente.i.ControlVigenciaVersionRol.ActoRef, fuente.i.ControlVigenciaVersionRol.MotivoCodigo = "acto_retirada", "retirada"
			case "campo_ajeno":
				fuente.i.VersionRol.Concesiones[len(fuente.i.VersionRol.Concesiones)-1].CamposPermitidos = []string{"datos_personales"}
			case "ambito_ajeno":
				p.soporte.unidadRef = "unidad:ajena"
			case "fuente_ausente":
				p.fuenteRespuestas = nil
			case "auditoria_fallida":
				registro.fallar = true
			}
			lecturas := 0
			ruta, err := nuevaRutaPoliticaCeseBolsa(p, func(context.Context) (bolsadomain.PoliticaCese, error) {
				lecturas++
				if caso.cambio == "lectura_fallida" {
					return bolsadomain.PoliticaCese{}, errors.New("fallo interno")
				}
				return bolsadomain.PoliticaCese{Version: 2, CatalogoRef: "catalogo:bolsa:cese:v2", CatalogoSHA256: strings.Repeat("a", 64),
					Mapeo: map[string]string{"interinidad|vacante": "general"}, MesesGeneral: 5, MesesAcumulacion: 9,
					Computo: "fecha_cese_meses_calendario_ajuste_fin_mes", Estado: "ejemplo_sintetico", PublicadaEn: ahora}, nil
			}, registro, "vec-rrhh")
			if err != nil || ruta.Ruta != bolsahttp.RutaPoliticaCese {
				t.Fatalf("montaje: %v", err)
			}
			w := httptest.NewRecorder()
			ruta.Manejador.ServeHTTP(w, httptest.NewRequest(http.MethodGet, ruta.Ruta, nil).WithContext(ctx))
			if w.Code != caso.codigo || lecturas != caso.lecturas || registro.llamadas != 1 {
				t.Fatalf("HTTP esperado=%d actual=%d; lecturas esperado=%d actual=%d; auditorías esperado=1 actual=%d; %s", caso.codigo, w.Code, caso.lecturas, lecturas, registro.llamadas, w.Body.String())
			}
			if w.Code == http.StatusOK {
				if !strings.Contains(w.Body.String(), `"version":2`) || registro.datos.Datos.RecursoRef != recursoAuditoriaPoliticaCeseBolsa ||
					registro.datos.Datos.Resultado != vecdomain.ResultadoIntentoAuditoriaExito || registro.datos.ResultadoContexto.Contexto.PersonaRef != actor.Resultado.Contexto.PersonaRef {
					t.Fatal("la política publicada no conserva constancia nominal")
				}
			} else if strings.Contains(w.Body.String(), "catalogo_sha256") {
				t.Fatal("se filtró política tras un fallo")
			}
		})
	}
}
