package cobertura

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/ports"
)

func TestPreparacionOperacionDecisionCoberturaPropietariaLigaTodoYCopia(t *testing.T) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	datos := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud, datos,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := preparacion.EstadoPara(solicitud)
	propiedad, errPropiedad := preparacion.DatosPropietariaPara(solicitud)
	if err != nil || errPropiedad != nil ||
		estado != PreparacionOperacionDecisionCoberturaPropietaria ||
		propiedad.ReservaRef != datos.ReservaRef ||
		propiedad.ReciboRef != datos.ReciboRef ||
		propiedad.ActuacionRef != datos.ActuacionRef ||
		propiedad.AuditoriaRef != datos.AuditoriaRef ||
		propiedad.EventoRef != datos.EventoRef ||
		propiedad.CorrelacionVECRef != datos.CorrelacionVECRef ||
		propiedad.DecisionVECRef != datos.DecisionVECRef ||
		propiedad.RevisionCercado != 1 ||
		!propiedad.PropiedadHasta.Equal(datos.PropiedadHasta) {
		t.Fatalf("reserva propietaria incompleta: %#v / %v", propiedad, err)
	}
	datos.AgregadoAnterior.Solicitud.Detalle = "adulterado fuera"
	propiedad.AgregadoAnterior.Solicitud.Detalle = "adulterado en copia"
	releida, err := preparacion.DatosPropietariaPara(solicitud)
	if err != nil ||
		releida.AgregadoAnterior.Solicitud.Detalle == "adulterado fuera" ||
		releida.AgregadoAnterior.Solicitud.Detalle == "adulterado en copia" {
		t.Fatal("la reserva compartió el agregado mutable")
	}
	if _, err := json.Marshal(preparacion); !errors.Is(
		err, ErrSerializacionOperacionDecisionCoberturaProhibida,
	) {
		t.Fatalf("preparación serializable: %v", err)
	}
}

func TestPreparacionOperacionDecisionCoberturaPropietariaExigeTokenExacto(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud,
		datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud),
	)
	if err != nil {
		t.Fatal(err)
	}
	otroToken, err := GenerarTokenPropietarioOperacionDecisionCobertura()
	if err != nil {
		t.Fatal(err)
	}
	otraSolicitud, err := NuevaSolicitudReservarOperacionDecisionCobertura(
		consulta, otroToken,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := preparacion.EstadoPara(otraSolicitud); !errors.Is(
		err, ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) {
		t.Fatalf("otro propietario heredó la reserva: %v", err)
	}
}

func TestPreparacionOperacionDecisionCoberturaOcupadaNoExponeEstadoAjeno(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	ambito, semantica, _ := consulta.parActivo()
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaOcupada(
		solicitud, ambito, semantica,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := preparacion.EstadoPara(solicitud)
	if err != nil || estado != PreparacionOperacionDecisionCoberturaOcupada {
		t.Fatalf("estado ocupado inválido: %q %v", estado, err)
	}
	if _, err := preparacion.DatosPropietariaPara(solicitud); err == nil {
		t.Fatal("ocupada expuso propietario, agregado o referencias")
	}
	if _, err := preparacion.ReciboConfirmadoPara(consulta); err == nil {
		t.Fatal("ocupada fabricó recibo")
	}
	texto := fmt.Sprintf("%v %#v", preparacion, preparacion)
	for _, prohibido := range []string{
		"reserva_", "recibo_", "actor_", "expediente_", "propiedad",
	} {
		if strings.Contains(texto, prohibido) {
			t.Fatalf("ocupada filtró %q en %q", prohibido, texto)
		}
	}
}

func TestPreparacionOperacionDecisionCoberturaConfirmadaSoloEntregaReciboLigado(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	reserva := reservaTerminalOperacionDecisionCoberturaPrueba(
		t,
		consulta,
		solicitud,
	)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaConfirmada(
		consulta, reserva, recibo,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := preparacion.EstadoPara(solicitud)
	releido, errRecibo := preparacion.ReciboConfirmadoPara(consulta)
	if err != nil || errRecibo != nil ||
		estado != PreparacionOperacionDecisionCoberturaConfirmada ||
		!reflect.DeepEqual(releido, recibo) {
		t.Fatalf("replay confirmado incoherente: %#v %v", releido, err)
	}
	if _, err := preparacion.DatosPropietariaPara(solicitud); err == nil {
		t.Fatal("confirmada expuso una reserva propietaria")
	}
	tipo := reflect.TypeOf(recibo)
	for _, nombre := range []string{
		"ActorRef", "PerfilRef", "OrganizacionRef", "ExpedienteRef",
		"Motivo", "PredecesoraRef", "TokenPropietarioSHA256",
	} {
		if _, existe := tipo.FieldByName(nombre); existe {
			t.Fatalf("recibo mínimo expone %s", nombre)
		}
	}
}

func TestReplayOperacionDecisionCoberturaConservaDenegacionVECComoTerminal(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	recibo := reciboDenegadoVECOperacionDecisionCoberturaPrueba(t, solicitud)
	reserva := reservaTerminalOperacionDecisionCoberturaPrueba(
		t,
		consulta,
		solicitud,
	)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaConfirmada(
		consulta, reserva, recibo,
	)
	if err != nil {
		t.Fatal(err)
	}
	estado, err := preparacion.EstadoPara(solicitud)
	releido, errRecibo := preparacion.ReciboConfirmadoPara(consulta)
	_, aplicada := releido.ResultadoAplicado()
	denegada, esDenegada := releido.ResultadoDenegadoVEC()
	if err != nil || errRecibo != nil ||
		estado != PreparacionOperacionDecisionCoberturaConfirmada ||
		aplicada || !esDenegada ||
		denegada != (ResultadoDenegadoVECOperacionDecisionCobertura{}) ||
		releido.ConcedidaVEC ||
		releido.CodigoProbatorioVEC != "accion_no_concedida" {
		t.Fatalf("denegación VEC no reproducible: %#v %v", releido, err)
	}
}

func TestReciboOperacionDecisionCoberturaExigeUnaSolaRamaTerminal(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	aplicado := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	sinRama := clonarReciboOperacionDecisionCobertura(aplicado)
	sinRama.Aplicada = nil
	conDos := clonarReciboOperacionDecisionCobertura(aplicado)
	conDos.DenegadaVEC = &ResultadoDenegadoVECOperacionDecisionCobertura{}
	for nombre, recibo := range map[string]ReciboOperacionDecisionCobertura{
		"sin rama":  sinRama,
		"dos ramas": conDos,
	} {
		t.Run(nombre, func(t *testing.T) {
			if recibo.ValidarPara(consulta) == nil {
				t.Fatal("unión terminal inválida aceptada")
			}
		})
	}
}

func TestReciboOperacionDecisionCoberturaCotejaReservaCongelada(t *testing.T) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	reserva := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	if err := recibo.ValidarParaReservaCongelada(
		solicitud, reserva,
	); err != nil {
		t.Fatalf("vínculos congelados válidos rechazados: %v", err)
	}
	casos := map[string]func(*ReciboOperacionDecisionCobertura){
		"reserva": func(r *ReciboOperacionDecisionCobertura) {
			r.ReservaRef = "reserva_decision_cobertura_otra"
		},
		"recibo": func(r *ReciboOperacionDecisionCobertura) {
			r.ReciboRef = "recibo_decision_cobertura_otro"
		},
		"auditoría": func(r *ReciboOperacionDecisionCobertura) {
			r.AuditoriaRef = "auditoria_decision_cobertura_otra"
		},
		"correlación VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.CorrelacionVECRef = "correlacion_vec_decision_cobertura_otra"
		},
		"decisión VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.DecisionVECRef = "decision_vec_autorizacion_otra"
		},
		"evento": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.EventoRef = "evento_decision_cobertura_otro"
		},
		"actuación": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.ActuacionRef = "actuacion_decision_cobertura_otra"
		},
		"cercado": func(r *ReciboOperacionDecisionCobertura) {
			r.RevisionCercado++
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			adulterado := clonarReciboOperacionDecisionCobertura(recibo)
			mutar(&adulterado)
			if adulterado.ValidarParaReservaCongelada(
				solicitud, reserva,
			) == nil {
				t.Fatal("vínculo congelado adulterado aceptado")
			}
		})
	}
}

func TestOperacionDecisionCoberturaDetectaColisionSemantica(t *testing.T) {
	identidad := identidadOperacionDecisionCoberturaPrueba()
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(t, identidad)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud,
		datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud),
	)
	if err != nil {
		t.Fatal(err)
	}
	sellosOriginales := sellosOperacionDecisionCoberturaPrueba(t)
	datosAmbitos, _ := sellosOriginales.AmbitosIdempotenciaHMAC.Datos()
	semanticaDistinta, err := ports.NuevaColeccionSellosHMAC(
		selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 2, "8",
		),
		[]string{selloOperacionDecisionCoberturaPrueba(
			dominioSemanticaOperacionDecisionCobertura, 1, "7",
		)},
	)
	if err != nil {
		t.Fatal(err)
	}
	identidad.viaElegida = "oferta_sae"
	ambitos, err := ports.NuevaColeccionSellosHMAC(
		datosAmbitos.Activo.Valor,
		[]string{datosAmbitos.Retenidos[0].Valor},
	)
	if err != nil {
		t.Fatal(err)
	}
	consultaColision, err :=
		NuevaSolicitudConsultarOperacionDecisionCoberturaConfirmada(
			identidad,
			SellosOperacionDecisionCobertura{
				AmbitosIdempotenciaHMAC: ambitos,
				HuellasSemanticasHMAC:   semanticaDistinta,
			},
		)
	if err != nil {
		t.Fatal(err)
	}
	token, _ := GenerarTokenPropietarioOperacionDecisionCobertura()
	solicitudColision, _ := NuevaSolicitudReservarOperacionDecisionCobertura(
		consultaColision, token,
	)
	if _, err := preparacion.EstadoPara(solicitudColision); !errors.Is(
		err, ErrOperacionDecisionCoberturaIdempotenteInvalida,
	) {
		t.Fatalf("clave reutilizada con otra semántica aceptada: %v", err)
	}
}

func TestOperacionDecisionCoberturaAceptaReplayDeGeneracionRetenida(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	sellos := sellosOperacionDecisionCoberturaPrueba(t)
	ambitos, _ := sellos.AmbitosIdempotenciaHMAC.Datos()
	semanticas, _ := sellos.HuellasSemanticasHMAC.Datos()
	datos := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	datos.AmbitoIdempotenciaHMAC = ambitos.Retenidos[0].Valor
	datos.HuellaSemanticaHMAC = semanticas.Retenidos[0].Valor
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud, datos,
	)
	if err != nil {
		t.Fatalf("generación retenida rechazada: %v", err)
	}
	if _, err := preparacion.DatosPropietariaPara(solicitud); err != nil {
		t.Fatal(err)
	}
	recibo := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	recibo.AmbitoIdempotenciaHMAC = ambitos.Retenidos[0].Valor
	recibo.HuellaSemanticaHMAC = semanticas.Retenidos[0].Valor
	if recibo.ValidarPara(consulta) != nil {
		t.Fatal("recibo de generación retenida rechazado")
	}
}

func TestReservaOperacionDecisionCoberturaCercaReapropiacionYAcotaLease(t *testing.T) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	base := datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud)
	casos := map[string]func(*DatosReservaPropietariaOperacionDecisionCobertura){
		"cercado repetido": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.RevisionCercado = d.RevisionCercadoAnterior
		},
		"salto de cercado": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.RevisionCercado = d.RevisionCercadoAnterior + 2
		},
		"desbordamiento": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.RevisionCercadoAnterior =
				MaximoEnteroSeguroOperacionDecisionCobertura
			d.RevisionCercado = 0
		},
		"lease excesivo": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.PropiedadHasta = d.ObservadaEnDB.Add(
				MaximoLeaseOperacionDecisionCobertura + time.Microsecond,
			)
		},
		"lease vacío": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.PropiedadHasta = d.ObservadaEnDB
		},
		"tiempo no UTC": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.ObservadaEnDB = d.ObservadaEnDB.In(
				time.FixedZone("local", 3600),
			)
		},
		"submicrosegundo": func(d *DatosReservaPropietariaOperacionDecisionCobertura) {
			d.PropiedadHasta = d.PropiedadHasta.Add(time.Nanosecond)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			datos := base
			mutar(&datos)
			if _, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
				solicitud, datos,
			); err == nil {
				t.Fatal("reserva insegura aceptada")
			}
		})
	}

	reapropiada := base
	reapropiada.RevisionCercadoAnterior = 41
	reapropiada.RevisionCercado = 42
	reapropiada.ObservadaEnDB = base.PropiedadHasta
	reapropiada.PropiedadHasta = reapropiada.ObservadaEnDB.Add(
		MaximoLeaseOperacionDecisionCobertura,
	)
	if _, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud, reapropiada,
	); err != nil {
		t.Fatalf("reapropiación cercada válida rechazada: %v", err)
	}
}

func TestReciboOperacionDecisionCoberturaRechazaAdulteracion(t *testing.T) {
	consulta, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	base := reciboOperacionDecisionCoberturaPrueba(t, solicitud)
	casos := map[string]func(*ReciboOperacionDecisionCobertura){
		"recibo": func(r *ReciboOperacionDecisionCobertura) {
			r.ReciboRef = ""
		},
		"reserva": func(r *ReciboOperacionDecisionCobertura) {
			r.ReservaRef = ""
		},
		"auditoría": func(r *ReciboOperacionDecisionCobertura) {
			r.AuditoriaRef = ""
		},
		"correlación VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.CorrelacionVECRef = ""
		},
		"decisión VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.DecisionVECRef = ""
		},
		"huella VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.DecisionVECHuellaSHA256 = strings.Repeat("0", 64)
		},
		"código VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = ""
		},
		"código VEC inventado": func(r *ReciboOperacionDecisionCobertura) {
			r.CodigoProbatorioVEC = "aceptada_por_rrhh"
		},
		"resultado VEC": func(r *ReciboOperacionDecisionCobertura) {
			r.ConcedidaVEC = false
		},
		"decisión": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.DecisionCoberturaRef = ""
		},
		"huella": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.DecisionCoberturaHuella = strings.Repeat("0", 64)
		},
		"versión": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.VersionResultante++
		},
		"evento": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.EventoRef = ""
		},
		"actuación": func(r *ReciboOperacionDecisionCobertura) {
			r.Aplicada.ActuacionRef = ""
		},
		"cercado": func(r *ReciboOperacionDecisionCobertura) {
			r.RevisionCercado = 0
		},
		"ámbito": func(r *ReciboOperacionDecisionCobertura) {
			r.AmbitoIdempotenciaHMAC = strings.Replace(
				r.AmbitoIdempotenciaHMAC, strings.Repeat("a", 64),
				strings.Repeat("6", 64), 1,
			)
		},
		"semántica": func(r *ReciboOperacionDecisionCobertura) {
			r.HuellaSemanticaHMAC = strings.Replace(
				r.HuellaSemanticaHMAC, strings.Repeat("c", 64),
				strings.Repeat("6", 64), 1,
			)
		},
		"instante": func(r *ReciboOperacionDecisionCobertura) {
			r.ConfirmadaEn = r.ConfirmadaEn.Add(time.Nanosecond)
		},
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			recibo := clonarReciboOperacionDecisionCobertura(base)
			mutar(&recibo)
			if recibo.ValidarPara(consulta) == nil {
				t.Fatal("recibo adulterado aceptado")
			}
		})
	}
}

func TestPreparacionOperacionDecisionCoberturaEsInmutableEnLecturaConcurrente(t *testing.T) {
	_, solicitud := solicitudReservaOperacionDecisionCoberturaPrueba(
		t, identidadOperacionDecisionCoberturaPrueba(),
	)
	preparacion, err := NuevaPreparacionOperacionDecisionCoberturaPropietaria(
		solicitud,
		datosPropiedadOperacionDecisionCoberturaPrueba(t, solicitud),
	)
	if err != nil {
		t.Fatal(err)
	}
	var grupo sync.WaitGroup
	errores := make(chan error, 64)
	for indice := 0; indice < 64; indice++ {
		grupo.Add(1)
		go func() {
			defer grupo.Done()
			estado, err := preparacion.EstadoPara(solicitud)
			if err != nil ||
				estado != PreparacionOperacionDecisionCoberturaPropietaria {
				errores <- errors.New("estado concurrente inválido")
				return
			}
			datos, err := preparacion.DatosPropietariaPara(solicitud)
			if err != nil || datos.AgregadoAnterior.Validar() != nil {
				errores <- errors.New("copia concurrente inválida")
			}
		}()
	}
	grupo.Wait()
	close(errores)
	for err := range errores {
		t.Fatal(err)
	}
}
