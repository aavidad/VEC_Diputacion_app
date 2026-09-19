package mibolsa

import (
	"context"
	"errors"
	"testing"
	"time"
	bolsa "vec-diputacion-granada/internal/modules/bolsa/ports"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func TestConsultaPropiaConPDPYMaterialNominalV3(t *testing.T) {
	e := nuevoEntorno(t)
	r, err := e.servicio.Consultar(context.Background(), e.orden)
	exigir(t, err)
	s := e.repositorio.solicitud
	if e.fuente.invocaciones != 1 || e.concesiones.invocaciones != 1 || e.emisor.llamadas != 1 || e.emisor.exportaciones != 1 || e.firmas != 1 || e.repositorio.llamadas != 1 || len(r.Participaciones) != 1 {
		t.Fatal("no recorrió una cadena completa")
	}
	if s.CandidatoRef != referenciaServicioContextoActorPrueba("can_", "c") || s.Material.ValidarEstructura() != nil {
		t.Fatal("selector o material inválidos")
	}
	datos, err := e.concesiones.orden.Datos()
	exigir(t, err)
	nominal, err := datos.Solicitud.Datos()
	exigir(t, err)
	if nominal.Accion != bolsa.AccionConsultarMiBolsa || nominal.Recurso.Referencia != "mi-bolsa:"+s.CandidatoRef || nominal.Recurso.Ambitos["candidato_ref"] != s.CandidatoRef || nominal.Finalidad != bolsa.FinalidadMiBolsa {
		t.Fatal("solicitud inexacta")
	}
}

func TestContextosNoAptosNoLleganAPDPNiMaterialNiRepositorio(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		opcion  opcionContexto
		cambiar func(*entornoMiBolsa)
	}{
		{nombre: "cero candidatos", opcion: func(i *domain.InstantaneaContextoActor) { i.Vinculos = nil }},
		{nombre: "varios candidatos", cambiar: func(e *entornoMiBolsa) {
			i := &e.orden.ResultadoContexto.Contexto.Instantanea
			v := i.Vinculos[0]
			v.VinculoRef = referenciaServicioContextoActorPrueba("vin_", "x")
			v.Referencia = referenciaServicioContextoActorPrueba("can_", "x")
			i.Vinculos = append(i.Vinculos, v)
		}},
		{nombre: "contexto alterado", cambiar: func(e *entornoMiBolsa) { e.orden.ResultadoContexto.HuellaSHA256 = "alterada" }},
		{nombre: "demo", cambiar: func(e *entornoMiBolsa) {
			e.orden.ResultadoContexto.Contexto.Principal.AuthMethod = domain.AuthMethodDemo
		}},
		{nombre: "superficie interna", cambiar: func(e *entornoMiBolsa) {
			a := e.autenticacion
			a.Superficie = domain.SuperficieAutenticacionInternaCorporativaV1
			v, err := domain.CrearVinculoAutenticacionActorV2(context.Background(), &revalidadorVinculoAplicacionAdversarial{resultado: a}, domain.SolicitudRevalidacionAutenticacionActorV1{AutenticacionRef: a.AutenticacionRef, SesionRef: a.SesionRef}, resolutorContextoAutorizacionV3Prueba{resultado: e.resultado}, solicitudServicioContextoActorPrueba(), &relojAutorizacionServicioPrueba{ahora: e.ahora})
			exigir(t, err)
			e.orden.Vinculo = v
		}},
		{nombre: "vinculo ausente", cambiar: func(e *entornoMiBolsa) { e.orden.Vinculo = domain.VinculoAutenticacionActorV2{} }},
		{nombre: "motivo ausente", cambiar: func(e *entornoMiBolsa) { e.orden.Motivo = domain.ReferenciaEntradaCatalogo{} }},
		{nombre: "correlacion ausente", cambiar: func(e *entornoMiBolsa) { e.orden.Correlacion = domain.ReferenciaCorrelacionAutorizacionV2{} }},
		{nombre: "caducado", cambiar: func(e *entornoMiBolsa) {
			e.servicio.reloj = &relojAutorizacionServicioPrueba{ahora: e.ahora.Add(time.Hour)}
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var opciones []opcionContexto
			if caso.opcion != nil {
				opciones = append(opciones, caso.opcion)
			}
			e := nuevoEntorno(t, opciones...)
			if caso.cambiar != nil {
				caso.cambiar(e)
			}
			_, err := e.servicio.Consultar(context.Background(), e.orden)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || e.emisor.llamadas != 0 || e.firmas != 0 || e.emisor.exportaciones != 0 || e.repositorio.llamadas != 0 {
				t.Fatalf("frontera atravesada: err=%v pdp=%d firmas=%d repo=%d", err, e.emisor.llamadas, e.firmas, e.repositorio.llamadas)
			}
		})
	}
}

func TestDenegacionOIndisponibilidadSinMaterialNiConsulta(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*entornoMiBolsa)
	}{
		{"sin concesion", func(e *entornoMiBolsa) {
			e.fuente.instantanea.VersionRol.Concesiones[0].Accion = "bolsa.otra.consultar"
		}},
		{"ambito ajeno", func(e *entornoMiBolsa) {
			e.fuente.instantanea.AsignacionPerfil.Ambitos[0].Valores = []string{referenciaServicioContextoActorPrueba("can_", "x")}
		}},
		{"fuente no disponible", func(e *entornoMiBolsa) { e.fuente.err = ports.ErrFuenteAutorizacionNoDisponible }},
		{"registro no disponible", func(e *entornoMiBolsa) { e.concesiones.err = ports.ErrRegistroDecisionNoDisponible }},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntorno(t)
			caso.cambiar(e)
			_, err := e.servicio.Consultar(context.Background(), e.orden)
			if !errors.Is(err, bolsa.ErrMaterialMiBolsaNoDisponible) || e.firmas != 0 || e.emisor.exportaciones != 0 || e.repositorio.llamadas != 0 {
				t.Fatalf("rechazo incorrecto: %v firmas=%d exportaciones=%d repo=%d", err, e.firmas, e.emisor.exportaciones, e.repositorio.llamadas)
			}
		})
	}
}

func TestDecisionRestringidaNoExportaNiConsulta(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*entornoMiBolsa)
	}{
		{"campos ausentes", func(e *entornoMiBolsa) { e.fuente.instantanea.VersionRol.Concesiones[0].CamposPermitidos = nil }},
		{"campos adicionales", func(e *entornoMiBolsa) {
			e.fuente.instantanea.VersionRol.Concesiones[0].CamposPermitidos = append(e.fuente.instantanea.VersionRol.Concesiones[0].CamposPermitidos, "otro")
		}},
		{"obligacion no implementada", func(e *entornoMiBolsa) {
			e.fuente.instantanea.VersionRol.Concesiones[0].Obligaciones = []string{"firma"}
		}},
		{"recurso cambiado", func(e *entornoMiBolsa) {
			e.emisor.cambiar = func(s domain.SolicitudAutorizacionLigadaV3) domain.SolicitudAutorizacionLigadaV3 {
				d, err := s.Datos()
				exigir(t, err)
				d.Recurso.Referencia = "mi-bolsa:otro"
				n, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
				exigir(t, err)
				return n
			}
		}},
		{"vence durante emision", func(e *entornoMiBolsa) {
			e.emisor.despues = func() { e.servicio.reloj = &relojAutorizacionServicioPrueba{ahora: e.ahora.Add(2 * time.Minute)} }
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntorno(t)
			caso.cambiar(e)
			_, err := e.servicio.Consultar(context.Background(), e.orden)
			if !errors.Is(err, domain.ErrAutorizacionDenegada) || e.emisor.llamadas != 1 || e.emisor.exportaciones != 0 || e.repositorio.llamadas != 0 {
				t.Fatalf("aceptó decisión no apta: %v", err)
			}
		})
	}
}

func TestConsultaVaciaErrorYResultadoInvalido(t *testing.T) {
	for _, caso := range []struct {
		nombre  string
		cambiar func(*entornoMiBolsa)
		causa   error
	}{
		{"vacia", func(e *entornoMiBolsa) {
			e.repositorio.mutar = func(r *bolsa.InstantaneaMiBolsa) { r.Participaciones = nil }
		}, nil},
		{"fallo repositorio", func(e *entornoMiBolsa) { e.repositorio.err = ports.ErrFuenteAutorizacionNoDisponible }, ports.ErrFuenteAutorizacionNoDisponible},
		{"orden mayor total", func(e *entornoMiBolsa) {
			e.repositorio.mutar = func(r *bolsa.InstantaneaMiBolsa) { r.Participaciones[0].OrdenInicial = 21 }
		}, bolsa.ErrResultadoMiBolsaInvalido},
		{"fecha distinta", func(e *entornoMiBolsa) {
			e.repositorio.mutar = func(r *bolsa.InstantaneaMiBolsa) { r.ConsultadaEn = r.ConsultadaEn.Add(time.Second) }
		}, bolsa.ErrResultadoMiBolsaInvalido},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			e := nuevoEntorno(t)
			caso.cambiar(e)
			r, err := e.servicio.Consultar(context.Background(), e.orden)
			if !errors.Is(err, caso.causa) || e.repositorio.llamadas != 1 {
				t.Fatalf("resultado: %v", err)
			}
			if err != nil && len(r.Participaciones) != 0 {
				t.Fatal("devolvió datos con error")
			}
		})
	}
}

func TestCancelacionAntesYDespuesDeEmision(t *testing.T) {
	for _, antes := range []bool{true, false} {
		e := nuevoEntorno(t)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if antes {
			cancel()
		} else {
			e.emisor.despues = cancel
		}
		_, err := e.servicio.Consultar(ctx, e.orden)
		if !errors.Is(err, context.Canceled) || e.repositorio.llamadas != 0 || e.emisor.exportaciones != 0 {
			t.Fatalf("cancelacion ignorada: %v", err)
		}
	}
}

func TestMaterialAjenoOCorruptoNoConsulta(t *testing.T) {
	for _, caso := range []string{"otro contexto", "decision alterada", "payload alterado", "version persona", "exportador falla", "material ausente", "proveedor falla"} {
		t.Run(caso, func(t *testing.T) {
			e := nuevoEntorno(t)
			if caso == "proveedor falla" {
				e.emisor.err = errors.New("fallo reservado")
			} else {
				e.emisor.mutarMaterial = func(m ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
					if caso == "exportador falla" {
						return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, errors.New("fallo reservado")
					}
					if caso == "material ausente" {
						return ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
					}
					contexto, decision, payload, version := m.ContextoActorCanonico(), m.DecisionCanonica(), m.PayloadVECAD3(), m.PersonaVersion()
					switch caso {
					case "otro contexto":
						contexto[10] ^= 1
					case "decision alterada":
						decision[10] ^= 1
					case "payload alterado":
						payload[len(payload)-1] ^= 1
					case "version persona":
						version++
					}
					return ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(m.CapacidadCanonica(), m.ResumenCapacidad(), decision, m.MotivoCanonico(), contexto, version, m.PerfilVersion(), payload, m.SobreCOSESign1(), m.EvidenciaVerificacion(), m.RaizPublicaSPKI())
				}
			}
			_, err := e.servicio.Consultar(context.Background(), e.orden)
			if err == nil || e.repositorio.llamadas != 0 {
				t.Fatalf("material ajeno alcanzó consulta: %v", err)
			}
			if caso == "proveedor falla" || caso == "exportador falla" {
				if !errors.Is(err, bolsa.ErrMaterialMiBolsaNoDisponible) {
					t.Fatal("fallo material no clasificado")
				}
			}
		})
	}
}
