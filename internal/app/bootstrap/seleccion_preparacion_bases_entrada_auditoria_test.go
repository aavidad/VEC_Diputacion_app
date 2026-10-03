package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	selhttp "vec-diputacion-granada/internal/modules/seleccion/adapters/http"
	core "vec-diputacion-granada/internal/vec/domain"
)

func TestPreparacionBasesHTTPEntradaInvalidaAuditaIdentidadAntesDeParser(t *testing.T) {
	for i := range 2 {
		for _, caso := range []string{"json", "tipo", "negocio"} {
			for _, disponible := range []bool{true, false} {
				t.Run(string(rune('A'+i))+caso+string(rune('0'+boolIndicePrueba(disponible))), func(t *testing.T) {
					b, ctx := brokerPreparacionBasesPrueba(t, i)
					z, _, err := b.contexto(ctx)
					if err != nil {
						t.Fatal(err)
					}
					// No se abre una transacción de negocio para estas entradas inválidas.
					servicio := &servicioIntentosPreparacionPrueba{cerrado: true}
					r := &registradorIntentosPreparacionPrueba{servicio: servicio}
					if !disponible {
						r.err = errors.New("fallo sintético de auditoría")
					}
					motivo := motivoCatalogoPlantillasCTDesarrollo()
					p, err := nuevoPreparadorBasesAuditadoV3(servicio, b, r, "vec-sintetico", motivo, motivo)
					if err != nil {
						t.Fatal(err)
					}
					b.registrarEntrada = func(ctx context.Context, z contextoSeguridadComunDesarrollo, j int, c core.ReferenciaCorrelacionAutorizacionV2, err error) error {
						return p.registrar(ctx, z, c, "", b.perfiles[j].accion, err)
					}
					h, err := selhttp.NuevaPreparacionBasesHandler(selhttp.ConfigPreparacionBases{Preparador: p, ResolverContexto: b.ResolverContextoHTTP,
						ValidarFrontera: func(*http.Request) error { return nil }})
					if err != nil {
						t.Fatal(err)
					}
					cuerpo := `{"modo":"actual","preparacion_ref":"preparacion:sintetica"}`
					if i == 0 {
						cuerpo = `{"esperada":{"preparacion_ref":"preparacion:sintetica","revision":0,"huella_material_sha256":""},"material":{"contenido":{},"referencias":[]},"clave_operacion":"operacion:sintetica"}`
					}
					esperado := 400
					tipo := "application/json"
					switch caso {
					case "json":
						cuerpo = `{"actor_ref":"cliente","actor_ref":"otra"}`
					case "tipo":
						tipo = "text/plain"
						esperado = 415
					case "negocio":
						if i == 0 {
							cuerpo = strings.Replace(cuerpo, `"revision":0`, `"revision":-1`, 1)
						} else {
							cuerpo = strings.Replace(cuerpo, `"actual"`, `"invalido"`, 1)
						}
					}
					peticion := httptest.NewRequest("POST", b.perfiles[i].ruta, strings.NewReader(cuerpo)).WithContext(ctx)
					peticion.Header.Set("Content-Type", tipo)
					peticion.Header.Set("X-Actor-Ref", "cliente")
					w := httptest.NewRecorder()
					h.ServeHTTP(w, peticion)
					if !disponible {
						esperado = 503
					}
					if w.Code != esperado || servicio.llamadas != 0 || len(r.ordenes) != 1 {
						t.Fatalf("HTTP=%d esperado=%d servicio=%d auditoría=%d", w.Code, esperado, servicio.llamadas, len(r.ordenes))
					}
					d := r.ordenes[0]
					if d.ResultadoContexto.HuellaSHA256 != z.Resultado.HuellaSHA256 || d.Datos.Resultado != core.ResultadoIntentoAuditoriaError ||
						d.Datos.Accion != b.perfiles[i].accion || !strings.HasPrefix(d.Datos.RecursoRef, "intento_") || strings.Contains(w.Body.String(), "cliente") {
						t.Fatal("entrada inválida perdió identidad o conservó datos del cliente")
					}
				})
			}
		}
	}
}

func boolIndicePrueba(b bool) int {
	if b {
		return 1
	}
	return 0
}
