package application

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/personal/domain"
	"vec-diputacion-granada/internal/modules/personal/ports"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

type autorizadorRelacionRPTPrueba func(context.Context, domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error)

func (f autorizadorRelacionRPTPrueba) AutorizarRelacionParaRPT(ctx context.Context, m domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
	return f(ctx, m)
}

type repositorioRelacionRPTPrueba func(context.Context, ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error)

func (f repositorioRelacionRPTPrueba) ConsultarRelacionParaRPT(ctx context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
	return f(ctx, o)
}

type intentosRelacionRPTPrueba struct {
	llamadas     []ports.IntentoLectorRelacionRPT
	err          error
	preflightErr error
	verificar    func(context.Context)
}

func (r *intentosRelacionRPTPrueba) VerificarRegistroRelacionRPT(context.Context) error {
	return r.preflightErr
}

func (r *intentosRelacionRPTPrueba) RegistrarIntentoRelacionRPT(ctx context.Context, i ports.IntentoLectorRelacionRPT) error {
	r.llamadas = append(r.llamadas, i)
	if r.verificar != nil {
		r.verificar(ctx)
	}
	return r.err
}

func consultaRelacionRPTPrueba(t *testing.T) ports.ConsultaRelacionParaRPTV1 {
	t.Helper()
	s := solicitudFichaPropiaPrueba(t, "pep_")
	return ports.ConsultaRelacionParaRPTV1{Actor: s.Actor, EmpleadoRef: "emp_" + strings.Repeat("e", 24), RelacionRef: "rel_" + strings.Repeat("r", 24), OrganismoRef: "organismo:sintetico", VersionEsperada: 2, Corte: s.Corte}
}

// Es un fixture estructural. No verifica COSE, HMAC, consumo V3 ni PostgreSQL.
func atestacionRelacionRPTPrueba(t *testing.T, m domain.MaterialLectorRelacionRPT, accion, audiencia string, ahora time.Time) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	h, err := m.HuellaSHA256()
	if err != nil {
		t.Fatal(err)
	}
	x, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_rpt_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_rpt_prueba", strings.Repeat("c", 64), accion, m.Recurso().Referencia, h, audiencia, ahora, ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, err := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	if err != nil {
		t.Fatal(err)
	}
	actor := m.Actor()
	canon, err := actor.RepresentacionCanonicaVinculadaV2()
	if err != nil {
		t.Fatal(err)
	}
	a, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), x, []byte("d"), []byte("m"), canon, actor.Instantanea.PersonaVersion, actor.Instantanea.PerfilVersion, []byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func resultadoRelacionRPTPrueba(o ports.OrdenLectorRelacionRPT) ports.ResultadoRelacionParaRPTV1 {
	s := o.Material.Solicitud()
	x := o.Autorizacion.ResumenCapacidad()
	return ports.ResultadoRelacionParaRPTV1{
		Relacion: ports.RelacionParaRPTV1{EmpleadoRef: s.EmpleadoRef, RelacionRef: s.RelacionRef, OrganismoRef: s.OrganismoRef, Version: s.VersionEsperada, Estado: "vigente", Periodo: ports.PeriodoPersonalNominalV1{Desde: "2024-01-01"}, Procedencia: ports.ProcedenciaPersonalNominalV1{ActoRef: "acto:sintetico", FuenteRef: "fuente:sintetica", FuenteVersion: "2", Certeza: ports.CertezaPersonalNoAcreditadaV1}},
		Corte:    s.Corte, Cobertura: ports.CoberturaPersonalNoAcreditadaV1,
		Evidencia: ports.EvidenciaRegistroEmpleadoB2{ReciboRef: "recibo:rpt", DecisionRef: x.DecisionRef(), EfectoRef: x.EfectoRef(), ConsumoHuellaSHA256: strings.Repeat("d", 64), AuditoriaRef: "auditoria:rpt", ConsultadaEn: x.EmitidaEn()},
	}
}

func servicioRelacionRPTPrueba(t *testing.T, in ports.ConsultaRelacionParaRPTV1, reloj *time.Time, repo repositorioRelacionRPTPrueba, intentos *intentosRelacionRPTPrueba) *ServicioLectorRelacionRPT {
	t.Helper()
	a := autorizadorRelacionRPTPrueba(func(_ context.Context, m domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
		if !reflect.DeepEqual(m.Actor(), in.Actor) {
			t.Fatal("se sustituyó el actor efectivo")
		}
		return atestacionRelacionRPTPrueba(t, m, ports.AccionRelacionParaRPTV1, ports.AudienciaRelacionParaRPTV1, *reloj), nil
	})
	s, err := NuevoServicioLectorRelacionRPT(a, repo, intentos, func() time.Time { return *reloj })
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func comprobarFalloRelacionRPT(t *testing.T, r ports.ResultadoRelacionParaRPTV1, err, esperado error, intentos *intentosRelacionRPTPrueba, in ports.ConsultaRelacionParaRPTV1, motivo string) {
	t.Helper()
	if !errors.Is(err, esperado) || !reflect.DeepEqual(r, ports.ResultadoRelacionParaRPTV1{}) {
		t.Fatalf("fallo con datos o error incorrecto: %+v, %v", r, err)
	}
	if len(intentos.llamadas) != 1 || intentos.llamadas[0].Motivo != motivo || intentos.llamadas[0].RelacionRef != in.RelacionRef || !reflect.DeepEqual(intentos.llamadas[0].Actor, in.Actor) {
		t.Fatalf("intento no conservado una vez: %+v", intentos.llamadas)
	}
}

func TestServicioRelacionRPTGestorConsultaEmpleadoAjenoConV3Nominal(t *testing.T) {
	for _, estado := range []string{"vigente", "suspendida", "finalizada"} {
		t.Run(estado, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			if in.Actor.Instantanea.Vinculos[0].Referencia == in.EmpleadoRef {
				t.Fatal("el fixture debe consultar otro empleado")
			}
			ahora := in.Actor.ResueltoEn
			intentos := &intentosRelacionRPTPrueba{}
			llamadas := 0
			repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				llamadas++
				x := o.Autorizacion.ResumenCapacidad()
				if x.Operacion() != ports.AccionRelacionParaRPTV1 || x.AudienciaConsumo() != ports.AudienciaRelacionParaRPTV1 || !o.Material.CoincideObjetivo(in.EmpleadoRef, in.RelacionRef, in.OrganismoRef, in.VersionEsperada) {
					t.Fatal("lectura sin autorización y selector RPT nominales")
				}
				r := resultadoRelacionRPTPrueba(o)
				r.Relacion.Estado = estado
				if estado == "finalizada" {
					r.Relacion.Periodo.Hasta = in.Corte.VigenteEn
				}
				return r, nil
			})
			r, err := servicioRelacionRPTPrueba(t, in, &ahora, repo, intentos).ConsultarRelacionParaRPT(context.Background(), in)
			if err != nil || llamadas != 1 || len(intentos.llamadas) != 0 || r.Relacion.Estado != estado || r.Relacion.Version != 2 || r.Relacion.Procedencia.Certeza != ports.CertezaPersonalNoAcreditadaV1 || r.Cobertura != ports.CoberturaPersonalNoAcreditadaV1 {
				t.Fatalf("hecho laboral alterado: %+v, %v", r, err)
			}
			if estado != "finalizada" && r.Relacion.Periodo.Hasta != "" {
				t.Fatal("se inventó fin para el período abierto")
			}
		})
	}
}

func TestServicioRelacionRPTNoReutilizaConcesionesDeOtraOperacion(t *testing.T) {
	for _, caso := range []struct {
		nombre, accion, audiencia string
		err                       error
		esperado                  error
	}{
		{"ausencia_permiso", "", "", domain.ErrLectorRelacionRPTDenegado, domain.ErrLectorRelacionRPTDenegado},
		{"pdp_caido", "", "", errors.New("detalle privado PDP"), domain.ErrLectorRelacionRPTNoDisponible},
		{"ficha_b2", domain.AccionFichaEmpleadoB2, domain.AudienciaFichaEmpleadoB2, nil, domain.ErrLectorRelacionRPTNoDisponible},
		{"ficha_propia", domain.AccionFichaPropia, domain.AudienciaFichaPropia, nil, domain.ErrLectorRelacionRPTNoDisponible},
		{"servicios_certificados", ports.AccionServiciosParaCertificadosV1, ports.AudienciaServiciosParaCertificadosV1, nil, domain.ErrLectorRelacionRPTNoDisponible},
		{"audiencia_ajena", ports.AccionRelacionParaRPTV1, domain.AudienciaFichaEmpleadoB2, nil, domain.ErrLectorRelacionRPTNoDisponible},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			intentos := &intentosRelacionRPTPrueba{}
			llamadas := 0
			a := autorizadorRelacionRPTPrueba(func(_ context.Context, m domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				if caso.err != nil {
					return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, caso.err
				}
				return atestacionRelacionRPTPrueba(t, m, caso.accion, caso.audiencia, in.Actor.ResueltoEn), nil
			})
			repo := repositorioRelacionRPTPrueba(func(context.Context, ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				llamadas++
				return ports.ResultadoRelacionParaRPTV1{}, nil
			})
			s, err := NuevoServicioLectorRelacionRPT(a, repo, intentos, func() time.Time { return in.Actor.ResueltoEn })
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.ConsultarRelacionParaRPT(context.Background(), in)
			motivo := "no_disponible"
			if errors.Is(caso.esperado, domain.ErrLectorRelacionRPTDenegado) {
				motivo = "denegado"
			}
			comprobarFalloRelacionRPT(t, r, err, caso.esperado, intentos, in, motivo)
			if llamadas != 0 || strings.Contains(err.Error(), "privado") {
				t.Fatal("lectura tras denegación o filtración del error interno")
			}
		})
	}
}

func TestServicioRelacionRPTRechazaRespuestaCruzadaOIncoherente(t *testing.T) {
	casos := map[string]func(*ports.ResultadoRelacionParaRPTV1){
		"empleado":             func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.EmpleadoRef = "emp_" + strings.Repeat("x", 24) },
		"relacion":             func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.RelacionRef = "rel_" + strings.Repeat("x", 24) },
		"organismo":            func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.OrganismoRef = "organismo:otro" },
		"version_antigua":      func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.Version = 1 },
		"corte_administrativo": func(r *ports.ResultadoRelacionParaRPTV1) { r.Corte.VigenteEn = "2026-09-24" },
		"corte_conocido":       func(r *ports.ResultadoRelacionParaRPTV1) { r.Corte.ConocidoEn = r.Corte.ConocidoEn.Add(-time.Second) },
		"certeza_acreditada": func(r *ports.ResultadoRelacionParaRPTV1) {
			r.Relacion.Procedencia.Certeza = ports.CertezaPersonalAcreditadaV1
		},
		"cobertura_completa": func(r *ports.ResultadoRelacionParaRPTV1) { r.Cobertura = ports.CoberturaPersonalCompletaV1 },
		"cobertura_vacia":    func(r *ports.ResultadoRelacionParaRPTV1) { r.Cobertura = "" },
		"estado":             func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.Estado = "desconocido" },
		"periodo":            func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.Periodo.Hasta = r.Relacion.Periodo.Desde },
		"sin_fuente":         func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.Procedencia.FuenteRef = "" },
		"version_fuente":     func(r *ports.ResultadoRelacionParaRPTV1) { r.Relacion.Procedencia.FuenteVersion = "02" },
		"decision_b2":        func(r *ports.ResultadoRelacionParaRPTV1) { r.Evidencia.DecisionRef = "decision:ficha_b2" },
		"efecto_b2":          func(r *ports.ResultadoRelacionParaRPTV1) { r.Evidencia.EfectoRef = r.Relacion.EmpleadoRef },
		"sin_auditoria":      func(r *ports.ResultadoRelacionParaRPTV1) { r.Evidencia.AuditoriaRef = "" },
		"sin_recibo":         func(r *ports.ResultadoRelacionParaRPTV1) { r.Evidencia.ReciboRef = "" },
		"huella":             func(r *ports.ResultadoRelacionParaRPTV1) { r.Evidencia.ConsumoHuellaSHA256 = "invalida" },
		"consulta_futura": func(r *ports.ResultadoRelacionParaRPTV1) {
			r.Evidencia.ConsultadaEn = r.Evidencia.ConsultadaEn.Add(time.Microsecond)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			ahora := in.Actor.ResueltoEn
			intentos := &intentosRelacionRPTPrueba{}
			repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				r := resultadoRelacionRPTPrueba(o)
				mutar(&r)
				return r, nil
			})
			r, err := servicioRelacionRPTPrueba(t, in, &ahora, repo, intentos).ConsultarRelacionParaRPT(context.Background(), in)
			comprobarFalloRelacionRPT(t, r, err, domain.ErrLectorRelacionRPTNoDisponible, intentos, in, "no_disponible")
		})
	}
}

func TestServicioRelacionRPTFuenteCaidaOPermisoRetiradoNoExponeDatos(t *testing.T) {
	for _, caso := range []struct {
		nombre        string
		err, esperado error
		motivo        string
	}{
		{"fuente_caida", errors.New("SQL con datos privados"), domain.ErrLectorRelacionRPTNoDisponible, "no_disponible"},
		{"permiso_retirado_en_fuente", domain.ErrLectorRelacionRPTDenegado, domain.ErrLectorRelacionRPTDenegado, "denegado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			ahora := in.Actor.ResueltoEn
			intentos := &intentosRelacionRPTPrueba{}
			repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				return resultadoRelacionRPTPrueba(o), caso.err
			})
			r, err := servicioRelacionRPTPrueba(t, in, &ahora, repo, intentos).ConsultarRelacionParaRPT(context.Background(), in)
			comprobarFalloRelacionRPT(t, r, err, caso.esperado, intentos, in, caso.motivo)
			if strings.Contains(err.Error(), "privados") {
				t.Fatal("se propagó detalle privado")
			}
		})
	}
}

func TestServicioRelacionRPTRevalidaActorYCapacidadTrasLectura(t *testing.T) {
	for _, caso := range []string{"actor_caduca", "capacidad_caduca"} {
		t.Run(caso, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			ahora := in.Actor.ResueltoEn
			if caso == "actor_caduca" {
				in.Actor.Instantanea.VigenteHasta = ahora.Add(time.Second)
			}
			intentos := &intentosRelacionRPTPrueba{}
			repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				r := resultadoRelacionRPTPrueba(o)
				if caso == "actor_caduca" {
					ahora = in.Actor.Instantanea.VigenteHasta
				} else {
					ahora = o.Autorizacion.ResumenCapacidad().ExpiraEn()
				}
				return r, nil
			})
			r, err := servicioRelacionRPTPrueba(t, in, &ahora, repo, intentos).ConsultarRelacionParaRPT(context.Background(), in)
			comprobarFalloRelacionRPT(t, r, err, domain.ErrLectorRelacionRPTDenegado, intentos, in, "denegado")
		})
	}
}

func TestServicioRelacionRPTAuditaSinHeredarCancelacionYFallaCerrado(t *testing.T) {
	for _, caso := range []string{"cancelado_antes", "cancelado_durante", "registro_caido"} {
		t.Run(caso, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			ahora := in.Actor.ResueltoEn
			type clavePrueba struct{}
			ctx, cancelar := context.WithCancel(context.WithValue(context.Background(), clavePrueba{}, "correlacion:sintetica"))
			defer cancelar()
			intentos := &intentosRelacionRPTPrueba{verificar: func(ctx context.Context) {
				if ctx.Err() != nil || ctx.Value(clavePrueba{}) != "correlacion:sintetica" {
					t.Fatal("auditoría heredó cancelación o perdió correlación")
				}
				if limite, ok := ctx.Deadline(); !ok || time.Until(limite) > 2*time.Second {
					t.Fatal("auditoría sin límite propio")
				}
			}}
			if caso == "cancelado_antes" {
				cancelar()
			}
			if caso == "registro_caido" {
				intentos.err = errors.New("destino privado caído")
			}
			llamadas := 0
			repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				llamadas++
				if caso == "registro_caido" {
					return resultadoRelacionRPTPrueba(o), domain.ErrLectorRelacionRPTDenegado
				}
				cancelar()
				return resultadoRelacionRPTPrueba(o), nil
			})
			r, err := servicioRelacionRPTPrueba(t, in, &ahora, repo, intentos).ConsultarRelacionParaRPT(ctx, in)
			esperado, motivo := error(context.Canceled), "no_disponible"
			if caso == "registro_caido" {
				esperado, motivo = domain.ErrLectorRelacionRPTNoDisponible, "denegado"
			}
			comprobarFalloRelacionRPT(t, r, err, esperado, intentos, in, motivo)
			if caso == "cancelado_antes" && llamadas != 0 {
				t.Fatal("fuente consultada con contexto cancelado")
			}
		})
	}
}

func TestServicioRelacionRPTConstructorExigeRegistroDeIntentos(t *testing.T) {
	var a autorizadorRelacionRPTPrueba
	var r repositorioRelacionRPTPrueba
	var i *intentosRelacionRPTPrueba
	reloj := func() time.Time { return time.Now() }
	for _, caso := range []struct {
		a     ports.ProveedorAutorizacionLectorRelacionRPT
		r     ports.RepositorioLectorRelacionRPT
		i     ports.RegistroIntentosLectorRelacionRPT
		reloj func() time.Time
	}{
		{a, nil, nil, reloj}, {nil, r, nil, reloj}, {nil, nil, i, reloj},
	} {
		if s, err := NuevoServicioLectorRelacionRPT(caso.a, caso.r, caso.i, caso.reloj); s != nil || !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
			t.Fatal("servicio incompleto compuesto")
		}
	}
	in := consultaRelacionRPTPrueba(t)
	ahora := in.Actor.ResueltoEn
	repo := repositorioRelacionRPTPrueba(func(_ context.Context, o ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
		return resultadoRelacionRPTPrueba(o), nil
	})
	s := servicioRelacionRPTPrueba(t, in, &ahora, repo, &intentosRelacionRPTPrueba{})
	if _, err := NuevoServicioLectorRelacionRPT(s.autorizador, repo, nil, reloj); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
		t.Fatal("registro opcional")
	}
	if _, err := NuevoServicioLectorRelacionRPT(s.autorizador, repo, i, reloj); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
		t.Fatal("registro tipado nulo aceptado")
	}
	if _, err := NuevoServicioLectorRelacionRPT(s.autorizador, repo, &intentosRelacionRPTPrueba{}, nil); !errors.Is(err, domain.ErrLectorRelacionRPTNoDisponible) {
		t.Fatal("reloj ausente aceptado")
	}
}

func TestServicioRelacionRPTVerificaDestinoDeIntentosAntesDeAutorizar(t *testing.T) {
	for _, registroCaido := range []bool{false, true} {
		in := consultaRelacionRPTPrueba(t)
		intentos := &intentosRelacionRPTPrueba{preflightErr: errors.New("fachada de intentos ausente")}
		if registroCaido {
			intentos.err = errors.New("registro de intentos ausente")
		}
		llamadasAutorizacion, llamadasFuente := 0, 0
		a := autorizadorRelacionRPTPrueba(func(context.Context, domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
			llamadasAutorizacion++
			return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
		})
		repo := repositorioRelacionRPTPrueba(func(context.Context, ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
			llamadasFuente++
			return ports.ResultadoRelacionParaRPTV1{}, nil
		})
		s, err := NuevoServicioLectorRelacionRPT(a, repo, intentos, func() time.Time { return in.Actor.ResueltoEn })
		if err != nil {
			t.Fatal(err)
		}
		r, err := s.ConsultarRelacionParaRPT(context.Background(), in)
		comprobarFalloRelacionRPT(t, r, err, domain.ErrLectorRelacionRPTNoDisponible, intentos, in, "no_disponible")
		if llamadasAutorizacion != 0 || llamadasFuente != 0 {
			t.Fatal("lectura o concesión antes de verificar la auditoría obligatoria")
		}
	}
}

func TestServicioRelacionRPTAuditaEntradaInvalidaOCorteFuturo(t *testing.T) {
	for _, caso := range []struct {
		nombre   string
		mutar    func(*ports.ConsultaRelacionParaRPTV1)
		esperado error
		motivo   string
	}{
		{"sin_version", func(in *ports.ConsultaRelacionParaRPTV1) { in.VersionEsperada = 0 }, domain.ErrLectorRelacionRPTInvalido, "entrada_invalida"},
		{"corte_futuro", func(in *ports.ConsultaRelacionParaRPTV1) { in.Corte.ConocidoEn = in.Actor.ResueltoEn.Add(time.Second) }, domain.ErrLectorRelacionRPTDenegado, "denegado"},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			caso.mutar(&in)
			ahora := in.Actor.ResueltoEn
			intentos := &intentosRelacionRPTPrueba{}
			autorizar, leer := 0, 0
			a := autorizadorRelacionRPTPrueba(func(context.Context, domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				autorizar++
				return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
			})
			repo := repositorioRelacionRPTPrueba(func(context.Context, ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				leer++
				return ports.ResultadoRelacionParaRPTV1{}, nil
			})
			s, err := NuevoServicioLectorRelacionRPT(a, repo, intentos, func() time.Time { return ahora })
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.ConsultarRelacionParaRPT(context.Background(), in)
			comprobarFalloRelacionRPT(t, r, err, caso.esperado, intentos, in, caso.motivo)
			if autorizar != 0 || leer != 0 {
				t.Fatal("entrada inválida alcanzó autorización o lectura")
			}
		})
	}
}

func TestServicioRelacionRPTRechazaAutorizacionDeOtroSelector(t *testing.T) {
	for _, caso := range []string{"sin_atestacion", "actor", "version", "empleado", "relacion", "organismo", "corte"} {
		t.Run(caso, func(t *testing.T) {
			in := consultaRelacionRPTPrueba(t)
			intentos := &intentosRelacionRPTPrueba{}
			a := autorizadorRelacionRPTPrueba(func(_ context.Context, m domain.MaterialLectorRelacionRPT) (vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3, error) {
				if caso == "sin_atestacion" {
					return vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, nil
				}
				s := m.Solicitud()
				switch caso {
				case "actor":
					s.Actor = solicitudFichaPropiaPrueba(t, "vin_").Actor
				case "version":
					s.VersionEsperada++
				case "empleado":
					s.EmpleadoRef = "emp_" + strings.Repeat("x", 24)
				case "relacion":
					s.RelacionRef = "rel_" + strings.Repeat("x", 24)
				case "organismo":
					s.OrganismoRef = "organismo:otro"
				case "corte":
					s.Corte.ConocidoEn = s.Corte.ConocidoEn.Add(-time.Second)
				}
				otro, err := domain.NuevoMaterialLectorRelacionRPT(s)
				if err != nil {
					t.Fatal(err)
				}
				return atestacionRelacionRPTPrueba(t, otro, ports.AccionRelacionParaRPTV1, ports.AudienciaRelacionParaRPTV1, in.Actor.ResueltoEn), nil
			})
			lecturas := 0
			repo := repositorioRelacionRPTPrueba(func(context.Context, ports.OrdenLectorRelacionRPT) (ports.ResultadoRelacionParaRPTV1, error) {
				lecturas++
				return ports.ResultadoRelacionParaRPTV1{}, nil
			})
			s, err := NuevoServicioLectorRelacionRPT(a, repo, intentos, func() time.Time { return in.Actor.ResueltoEn })
			if err != nil {
				t.Fatal(err)
			}
			r, err := s.ConsultarRelacionParaRPT(context.Background(), in)
			comprobarFalloRelacionRPT(t, r, err, domain.ErrLectorRelacionRPTNoDisponible, intentos, in, "no_disponible")
			if lecturas != 0 {
				t.Fatal("concesión de otro selector alcanzó la fuente")
			}
		})
	}
}
