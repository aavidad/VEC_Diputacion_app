package postgres

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

func materialReservaUsoRPTPrueba() ports.MaterialReservaUsoCategoriaRPT {
	return ports.MaterialReservaUsoCategoriaRPT{
		Consumidor: "contratacion_temporal", UsoRef: "uso:ct:001", CategoriaID: "categoria.uno",
		Publicacion: ports.ReferenciaPublicacionRPT{CatalogoID: descriptorRPTPrueba.CatalogoID,
			Version: 1, HuellaSHA256: strings.Repeat("a", 64)},
		ReservaReciboRef: "recibo:reserva:001",
	}
}

func TestReferenciasOpacasUsoRPTRespetanBytesUTF8SinRecortar(t *testing.T) {
	for _, valor := range []string{"uso", " a ", "a\nb", strings.Repeat("é", 80)} {
		if !referenciaUsoRPTValida(valor) {
			t.Fatalf("referencia opaca válida recortada: %q", valor)
		}
	}
	for _, valor := range []string{"ab", strings.Repeat("x", 161), strings.Repeat("é", 81), "a\x00b", string([]byte{'u', 's', 0xff})} {
		if referenciaUsoRPTValida(valor) {
			t.Fatalf("referencia fuera de octetos/UTF-8 admitida: %q", valor)
		}
	}
}

func TestUsoRefRPTDebePoderSerRecursoV3SinRestringirRecibos(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	inicio := &iniciadorUsoRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	gestor, err := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
	if err != nil {
		t.Fatal(err)
	}
	for _, usoRef := range []string{" a ", "a\nb", strings.Repeat("é", 80), "uso*ct", strings.Repeat("x", 161)} {
		malo := m
		malo.UsoRef = usoRef
		_, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{Material: malo})
		if !errors.Is(err, ports.ErrUsoCategoriaRPTInvalido) || inicio.llamadas != 0 {
			t.Fatalf("uso_ref incompatible con recurso V3 llegó a transacción: %q, %v", usoRef, err)
		}
	}
	for _, recibo := range []string{" a ", "a\nb", strings.Repeat("é", 80)} {
		opaco := m
		opaco.ReservaReciboRef = recibo
		if _, err := gestor.materialReserva(opaco); err != nil {
			t.Fatalf("recibo opaco rechazado: %q, %v", recibo, err)
		}
		if _, err := gestor.materialTerminal(ports.MaterialTerminalUsoCategoriaRPT{
			Reserva: opaco, TerminalReciboRef: "recibo:terminal:001", EvidenciaRef: recibo,
			EvidenciaSHA256: strings.Repeat("a", 64),
		}); err != nil {
			t.Fatalf("evidencia opaca rechazada: %q, %v", recibo, err)
		}
	}
}

func autorizacionUsoRPTPrueba(t *testing.T, accion, audiencia, usoRef string) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	return autorizacionUsoRPTHuellaPrueba(t, accion, audiencia, usoRef, strings.Repeat("a", 64))
}

func autorizacionUsoRPTHuellaPrueba(t *testing.T, accion, audiencia, usoRef, materialSHA256 string) (domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) {
	t.Helper()
	escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
	d, err := escenario.solicitud.Datos()
	if err != nil {
		t.Fatal(err)
	}
	d.Accion, d.Finalidad = accion, finalidadUsosRPT
	d.Recurso = domain.RecursoAutorizable{
		Referencia: usoRef, ModuloID: descriptorRPTPrueba.ModuloID, Tipo: tipoUsoRPT,
		Ambitos: map[string]string{"catalogo_id": descriptorRPTPrueba.CatalogoID,
			"modulo_id": descriptorRPTPrueba.ModuloID, "consumidor": "contratacion_temporal"},
		Atributos: map[string]string{"material_sha256": materialSHA256},
	}
	solicitud, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
	if err != nil {
		t.Fatal(err)
	}
	huella, err := d.Recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	huellaSolicitud, err := domain.HuellaSHA256SolicitudAutorizacionV3(solicitud)
	if err != nil {
		t.Fatal(err)
	}
	motivoCanonico, err := domain.RepresentacionCanonicaMotivoAutorizacionV2(d.ReferenciaMotivo)
	if err != nil {
		t.Fatal(err)
	}
	vinculo, err := d.VinculoAutenticacionActor.Datos()
	if err != nil {
		t.Fatal(err)
	}
	correlacion, err := d.Correlacion.ValorCanonico()
	if err != nil {
		t.Fatal(err)
	}
	decisionCanonica := jsonRPTPrueba(t, decisionLigaduraUsoRPT{
		Esquema:     domain.EsquemaHuellaDecisionAutorizacionV3,
		DecisionRef: "decision:rpt:uso:prueba", SolicitudHuellaSHA256: huellaSolicitud,
		MotivoHuellaSHA256: huellaBytesUsoRPT(motivoCanonico), ContextoRecursoHuellaSHA256: huella,
		CorrelacionRef: correlacion,
		PrincipalID:    vinculo.PrincipalID, PerfilActivoRef: vinculo.PerfilActivoRef,
	})
	contextoCanonico := escenario.resultado.RepresentacionCanonica
	resumen, err := ports.NuevoResumenCapacidadAtestacionAutorizacionV3(
		"decision:rpt:uso:prueba", huellaBytesUsoRPT(decisionCanonica), huellaBytesUsoRPT(motivoCanonico),
		vinculo.RegistroContextoRef, huellaBytesUsoRPT(contextoCanonico), accion, usoRef, huella,
		audiencia, escenario.ahora, escenario.ahora.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	raiz, _ := hex.DecodeString("302a300506032b65700321002152f8d19b791d24453242e15f2eab6cb7cffa7b6a5ed30097960e069881db12")
	autorizacion, err := ports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(
		bytes.Repeat([]byte("x"), 512), resumen, decisionCanonica, motivoCanonico, contextoCanonico, 1, 1,
		[]byte("p"), []byte("s"), []byte("e"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return solicitud, autorizacion
}

func TestLecturaDecisionLigaduraUsoRPTRechazaDuplicadosTiposYCola(t *testing.T) {
	_, exportacion := autorizacionUsoRPTPrueba(t, accionReservarUsoRPT, audienciaUsosRPT, materialReservaUsoRPTPrueba().UsoRef)
	canon := exportacion.DecisionCanonica()
	if _, err := leerDecisionLigaduraUsoRPT(canon); err != nil {
		t.Fatal("decisión de prueba íntegra rechazada")
	}
	for _, caso := range []struct {
		nombre string
		bytes  []byte
	}{
		{"clave duplicada", append(append([]byte(nil), canon[:len(canon)-1]...), []byte(`,"solicitud_huella_sha256":"`+strings.Repeat("f", 64)+`"}`)...)},
		{"tipo numérico", []byte(`{"solicitud_huella_sha256":123}`)},
		{"cola JSON", append(append([]byte(nil), canon...), []byte(` {}`)...)},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			if _, err := leerDecisionLigaduraUsoRPT(caso.bytes); err == nil {
				t.Fatal("decisión ambigua aceptada")
			}
		})
	}
}

func TestUsoRPTDeniegaSolicitudBAunqueRecursoYMaterialCoincidanConExportacionA(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	terminal := ports.MaterialTerminalUsoCategoriaRPT{Reserva: m, TerminalReciboRef: "recibo:terminal:001",
		EvidenciaRef: "evidencia:ct:001", EvidenciaSHA256: strings.Repeat("e", 64)}
	for _, accion := range []string{accionReservarUsoRPT, accionConfirmarUsoRPT, accionCancelarUsoRPT} {
		t.Run(accion, func(t *testing.T) {
			solicitudA, exportacionA := autorizacionUsoRPTPrueba(t, accion, audienciaUsosRPT, m.UsoRef)
			base, err := solicitudA.Datos()
			if err != nil {
				t.Fatal(err)
			}
			vinculoA, err := base.VinculoAutenticacionActor.Datos()
			if err != nil {
				t.Fatal(err)
			}
			escenario := nuevoEscenarioRegistroContextoActorV3PostgreSQLPrueba(t, true)
			autenticacionB := vinculoA.Autenticacion()
			autenticacionB.SesionRef = "ses_bbbbbbbbbbbbbbbbbbbbbb"
			actor := escenario.resultado.Contexto
			cuenta := domain.CuentaAutenticadaContextoActor{CuentaRef: actor.Instantanea.CuentaRef,
				Metodo: actor.Principal.AuthMethod, Garantia: actor.Principal.AuthAssurance}
			vinculoB, err := domain.CrearVinculoAutenticacionActorV2(t.Context(),
				revalidadorRegistroContextoActorV3PostgreSQLPrueba{autenticacionB},
				domain.SolicitudRevalidacionAutenticacionActorV1{
					AutenticacionRef: autenticacionB.AutenticacionRef, SesionRef: autenticacionB.SesionRef},
				resolutorRegistroContextoActorV3PostgreSQLPrueba{escenario.resultado},
				domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actor.PerfilActivoRef},
				relojRegistroContextoActorV3PostgreSQLPrueba{escenario.ahora})
			if err != nil {
				t.Fatal(err)
			}
			instantaneaB := actor.Instantanea
			instantaneaB.PerfilActivoRef = "prf_bbbbbbbbbbbbbbbbbbbbbb"
			instantaneaB.VinculoRef = "vca_bbbbbbbbbbbbbbbbbbbbbb"
			actorB, err := domain.NuevoContextoActor(cuenta, instantaneaB, actor.ResueltoEn)
			if err != nil {
				t.Fatal(err)
			}
			resultadoB := escenario.resultado
			resultadoB.Contexto = actorB
			resultadoB.RegistroContextoRef = "rca_bbbbbbbbbbbbbbbbbbbbbbbb"
			resultadoB.RepresentacionCanonica, err = actorB.RepresentacionCanonicaVinculadaV2()
			if err != nil {
				t.Fatal(err)
			}
			resultadoB.HuellaSHA256, err = actorB.HuellaSHA256VinculadaV2()
			if err != nil {
				t.Fatal(err)
			}
			manifiestoB, err := domain.RehidratarManifiestoProcedenciaContextoActorV1(escenario.resultado.ManifiestoProcedenciaCanonico)
			if err != nil {
				t.Fatal(err)
			}
			manifiestoB.Perfil.PerfilRef = instantaneaB.PerfilActivoRef
			manifiestoB.Contexto.VinculoRef = instantaneaB.VinculoRef
			resultadoB.ManifiestoProcedenciaCanonico, err = manifiestoB.RepresentacionCanonicaV1()
			if err != nil {
				t.Fatal(err)
			}
			resultadoB.ManifiestoProcedenciaHuellaSHA256, err = domain.HuellaSHA256ManifiestoProcedenciaContextoActorV1(resultadoB.ManifiestoProcedenciaCanonico)
			if err != nil {
				t.Fatal(err)
			}
			vinculoPerfilB, err := domain.CrearVinculoAutenticacionActorV2(t.Context(),
				revalidadorRegistroContextoActorV3PostgreSQLPrueba{vinculoA.Autenticacion()},
				domain.SolicitudRevalidacionAutenticacionActorV1{
					AutenticacionRef: autenticacionB.AutenticacionRef, SesionRef: vinculoA.SesionRef},
				resolutorRegistroContextoActorV3PostgreSQLPrueba{resultadoB},
				domain.SolicitudContextoActor{Cuenta: cuenta, PerfilActivoRef: actorB.PerfilActivoRef},
				relojRegistroContextoActorV3PostgreSQLPrueba{escenario.ahora})
			if err != nil {
				t.Fatal(err)
			}
			correlacionB, err := domain.GenerarReferenciaCorrelacionAutorizacionV2(t.Context(),
				generadorCorrelacionRegistroContextoActorV3PostgreSQLPrueba{valor: "correlacion_22222222222222222222222222222222"})
			if err != nil {
				t.Fatal(err)
			}
			for _, variante := range []struct {
				nombre  string
				cambiar func(*domain.DatosSolicitudAutorizacionLigadaV3)
			}{
				{"vinculo_actor_sesion", func(d *domain.DatosSolicitudAutorizacionLigadaV3) { d.VinculoAutenticacionActor = vinculoB }},
				{"perfil_contexto", func(d *domain.DatosSolicitudAutorizacionLigadaV3) { d.VinculoAutenticacionActor = vinculoPerfilB }},
				{"motivo", func(d *domain.DatosSolicitudAutorizacionLigadaV3) {
					d.ReferenciaMotivo.EntradaClave = "motivo_22222222222222222222222222222222"
				}},
				{"correlacion", func(d *domain.DatosSolicitudAutorizacionLigadaV3) { d.Correlacion = correlacionB }},
			} {
				t.Run(variante.nombre, func(t *testing.T) {
					d := base
					variante.cambiar(&d)
					solicitudB, err := domain.NuevaSolicitudAutorizacionLigadaV3(d)
					if err != nil {
						t.Fatal(err)
					}
					if d.Recurso.Referencia != base.Recurso.Referencia ||
						d.Recurso.Atributos["material_sha256"] != base.Recurso.Atributos["material_sha256"] {
						t.Fatal("caso A/B no conserva recurso y material")
					}
					inicio := &iniciadorUsoRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
					g, err := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
					if err != nil {
						t.Fatal(err)
					}
					var resultado ports.ResultadoUsoCategoriaRPT
					switch accion {
					case accionReservarUsoRPT:
						resultado, err = g.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{Material: m, Solicitud: solicitudB, Autorizacion: exportacionA})
					case accionConfirmarUsoRPT:
						resultado, err = g.ConfirmarUsoCategoriaRPT(t.Context(), ports.OrdenConfirmacionUsoCategoriaRPT{Material: terminal, Solicitud: solicitudB, Autorizacion: exportacionA})
					case accionCancelarUsoRPT:
						resultado, err = g.CancelarUsoCategoriaRPT(t.Context(), ports.OrdenCancelacionUsoCategoriaRPT{Material: terminal, Solicitud: solicitudB, Autorizacion: exportacionA})
					}
					if !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || resultado.Encontrado || inicio.llamadas != 0 {
						t.Fatalf("solicitud B accedió a PG con exportación A: %+v, %v, BeginTx=%d", resultado, err, inicio.llamadas)
					}
				})
			}
		})
	}
}

func reciboUsoRPTPrueba(t *testing.T, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3,
	m ports.MaterialReservaUsoCategoriaRPT, estado, reciboRef, terminalRef string,
) []byte {
	t.Helper()
	z := a.ResumenCapacidad()
	uso := usoCategoriaRPTWire{
		Consumidor: m.Consumidor, UsoRef: m.UsoRef, CategoriaID: m.CategoriaID,
		CatalogoID: m.Publicacion.CatalogoID, Version: m.Publicacion.Version,
		HuellaSHA256: m.Publicacion.HuellaSHA256, Estado: estado, Revision: 1,
		ReservaReciboRef: m.ReservaReciboRef, ReservadoEn: z.EmitidaEn(),
	}
	if terminalRef != "" {
		fecha := z.EmitidaEn().Add(time.Second)
		uso.Revision, uso.TerminalReciboRef, uso.TerminalEn = 2, &terminalRef, &fecha
	}
	return jsonRPTPrueba(t, reciboUsoRPTWire{
		DecisionRef: z.DecisionRef(), EfectoRef: z.EfectoRef(),
		HuellaEfectoSHA256: z.EfectoHuellaSHA256(), ConsumoHuellaSHA256: strings.Repeat("d", 64),
		AuditoriaRef: "auditoria:rpt:uso:prueba", ConsumidaEn: z.EmitidaEn().Add(time.Second),
		ConsumoNuevo: true, ReciboRef: reciboRef, Uso: jsonRPTPrueba(t, uso),
	})
}

type transaccionUsoRPTPrueba struct {
	*transaccionLecturaRPTPrueba
	pasos       []string
	errorCommit error
}

func (t *transaccionUsoRPTPrueba) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	t.pasos = append(t.pasos, "configurar")
	return t.transaccionLecturaRPTPrueba.Exec(ctx, sql, args...)
}

func (t *transaccionUsoRPTPrueba) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if sql == consultaHuellaMaterialRPT {
		t.pasos = append(t.pasos, "huella")
	} else {
		t.pasos = append(t.pasos, "fachada")
	}
	return t.transaccionLecturaRPTPrueba.QueryRow(ctx, sql, args...)
}

func (t *transaccionUsoRPTPrueba) Commit(ctx context.Context) error {
	t.pasos = append(t.pasos, "commit")
	if t.errorCommit != nil {
		return t.errorCommit
	}
	return t.transaccionLecturaRPTPrueba.Commit(ctx)
}

type iniciadorUsoRPTPrueba struct {
	tx       pgx.Tx
	opciones pgx.TxOptions
	llamadas int
}

func (i *iniciadorUsoRPTPrueba) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	i.opciones = opciones
	return i.tx, nil
}

func TestUsoRPTReservaEnUnaTransaccionYConservaTerminalEnReplay(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	s, a := autorizacionUsoRPTPrueba(t, accionReservarUsoRPT, audienciaUsosRPT, m.UsoRef)
	for _, caso := range []struct {
		estado      string
		terminalRef string
	}{
		{"reservado", ""}, {"confirmado", "recibo:terminal:001"}, {"cancelado", "recibo:terminal:001"},
	} {
		t.Run(caso.estado, func(t *testing.T) {
			tx := &transaccionUsoRPTPrueba{transaccionLecturaRPTPrueba: &transaccionLecturaRPTPrueba{
				respuesta: reciboUsoRPTPrueba(t, a, m, caso.estado, m.ReservaReciboRef, caso.terminalRef),
			}}
			inicio := &iniciadorUsoRPTPrueba{tx: tx}
			gestor, err := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
			if err != nil {
				t.Fatal(err)
			}
			resultado, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
				Material: m, Solicitud: s, Autorizacion: a,
			})
			if err != nil || !resultado.Encontrado || resultado.Uso == nil ||
				resultado.Uso.Estado != caso.estado || resultado.Uso.ReservaReciboRef != m.ReservaReciboRef ||
				resultado.Uso.Publicacion != m.Publicacion || !resultado.Evidencia.ConsumoNuevo || !tx.confirmada ||
				inicio.opciones.IsoLevel != pgx.Serializable || inicio.opciones.AccessMode != pgx.ReadWrite ||
				!tx.configurada || !reflect.DeepEqual(tx.pasos, []string{"configurar", "huella", "fachada", "commit"}) ||
				tx.consulta != consultaReservarUsoRPT || len(tx.argumentos) != 11 {
				t.Fatalf("reserva o replay no íntegros: %+v %v tx=%+v", resultado, err, tx)
			}
			var material map[string]any
			if json.Unmarshal([]byte(tx.argumentos[0].(string)), &material) != nil || len(material) != 8 ||
				material["catalogo_id"] != descriptorRPTPrueba.CatalogoID ||
				material["modulo_id"] != descriptorRPTPrueba.ModuloID || material["consumidor"] != m.Consumidor ||
				material["uso_ref"] != m.UsoRef || material["categoria_id"] != m.CategoriaID ||
				material["version"] != float64(m.Publicacion.Version) ||
				material["huella_sha256"] != m.Publicacion.HuellaSHA256 ||
				material["reserva_recibo_ref"] != m.ReservaReciboRef ||
				tx.argumentos[5] != strconv.FormatUint(a.PersonaVersion(), 10) {
				t.Fatalf("material de reserva no exacto: %+v", material)
			}
		})
	}
}

func TestUsoRPTTerminalExigeEstadoReciboYMaterial11Claves(t *testing.T) {
	m := ports.MaterialTerminalUsoCategoriaRPT{Reserva: materialReservaUsoRPTPrueba(),
		TerminalReciboRef: "recibo:terminal:001", EvidenciaRef: "evidencia:ct:001", EvidenciaSHA256: strings.Repeat("e", 64)}
	for _, caso := range []struct {
		accion string
		estado string
		correr func(*GestorUsosCategoriaRPTPostgreSQL, context.Context, ports.MaterialTerminalUsoCategoriaRPT,
			domain.SolicitudAutorizacionLigadaV3, ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoUsoCategoriaRPT, error)
	}{
		{accionConfirmarUsoRPT, "confirmado", func(g *GestorUsosCategoriaRPTPostgreSQL, ctx context.Context, m ports.MaterialTerminalUsoCategoriaRPT,
			s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoUsoCategoriaRPT, error) {
			return g.ConfirmarUsoCategoriaRPT(ctx, ports.OrdenConfirmacionUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
		}},
		{accionCancelarUsoRPT, "cancelado", func(g *GestorUsosCategoriaRPTPostgreSQL, ctx context.Context, m ports.MaterialTerminalUsoCategoriaRPT,
			s domain.SolicitudAutorizacionLigadaV3, a ports.ExportacionMaterialConsumoAutorizacionAtestadaV3) (ports.ResultadoUsoCategoriaRPT, error) {
			return g.CancelarUsoCategoriaRPT(ctx, ports.OrdenCancelacionUsoCategoriaRPT{Material: m, Solicitud: s, Autorizacion: a})
		}},
	} {
		t.Run(caso.estado, func(t *testing.T) {
			s, a := autorizacionUsoRPTPrueba(t, caso.accion, audienciaUsosRPT, m.Reserva.UsoRef)
			tx := &transaccionLecturaRPTPrueba{respuesta: reciboUsoRPTPrueba(t, a, m.Reserva, caso.estado,
				m.TerminalReciboRef, m.TerminalReciboRef)}
			inicio := &iniciadorUsoRPTPrueba{tx: tx}
			gestor, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Reserva.Consumidor)
			resultado, err := caso.correr(gestor, t.Context(), m, s, a)
			if err != nil || resultado.Uso == nil || resultado.Uso.Estado != caso.estado || !tx.confirmada ||
				len(tx.argumentos) != 11 {
				t.Fatalf("terminal no íntegro: %+v %v tx=%+v", resultado, err, tx)
			}
			var material map[string]any
			if json.Unmarshal([]byte(tx.argumentos[0].(string)), &material) != nil || len(material) != 11 ||
				material["terminal_recibo_ref"] != m.TerminalReciboRef ||
				material["evidencia_ref"] != m.EvidenciaRef ||
				material["evidencia_sha256"] != m.EvidenciaSHA256 {
				t.Fatalf("material terminal no exacto: %+v", material)
			}
			tx.respuesta = reciboUsoRPTPrueba(t, a, m.Reserva, "reservado", m.TerminalReciboRef, "")
			tx.confirmada = false
			if _, err := caso.correr(gestor, t.Context(), m, s, a); !errors.Is(err, ports.ErrUsoCategoriaRPTNoConfiable) || tx.confirmada || !tx.revertida {
				t.Fatalf("estado terminal falso llegó a COMMIT: %v tx=%+v", err, tx)
			}
		})
	}
}

func TestUsoRPTDeniegaAutoridadAjenaYRevierteRespuestaManipulada(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	s, a := autorizacionUsoRPTPrueba(t, accionReservarUsoRPT, audienciaUsosRPT, m.UsoRef)
	for _, caso := range []struct {
		nombre string
		mutar  func(*reciboUsoRPTWire)
	}{
		{"recibo ajeno", func(r *reciboUsoRPTWire) { r.ReciboRef = "recibo:ajeno" }},
		{"sin nuevo consumo V3", func(r *reciboUsoRPTWire) { r.ConsumoNuevo = false }},
		{"uso de otra publicación", func(r *reciboUsoRPTWire) {
			var uso usoCategoriaRPTWire
			_ = json.Unmarshal(r.Uso, &uso)
			uso.HuellaSHA256 = strings.Repeat("f", 64)
			r.Uso = jsonRPTPrueba(t, uso)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			var recibo reciboUsoRPTWire
			_ = json.Unmarshal(reciboUsoRPTPrueba(t, a, m, "reservado", m.ReservaReciboRef, ""), &recibo)
			caso.mutar(&recibo)
			tx := &transaccionLecturaRPTPrueba{respuesta: jsonRPTPrueba(t, recibo)}
			gestor, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
			if _, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
				Material: m, Solicitud: s, Autorizacion: a,
			}); !errors.Is(err, ports.ErrUsoCategoriaRPTNoConfiable) || tx.confirmada || !tx.revertida {
				t.Fatalf("recibo manipulado aceptado: %v tx=%+v", err, tx)
			}
		})
	}
	// La lectura del canon histórico necesita otro permiso AD3: un objeto
	// publicación extra en esta respuesta no está autorizado por [recibo,uso].
	var respuesta map[string]any
	if err := json.Unmarshal(reciboUsoRPTPrueba(t, a, m, "reservado", m.ReservaReciboRef, ""), &respuesta); err != nil {
		t.Fatal(err)
	}
	respuesta["publicacion"] = map[string]any{"catalogo_id": m.Publicacion.CatalogoID}
	txExtra := &transaccionLecturaRPTPrueba{respuesta: jsonRPTPrueba(t, respuesta)}
	gestorExtra, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: txExtra}, descriptorRPTPrueba, m.Consumidor)
	if _, err := gestorExtra.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
		Material: m, Solicitud: s, Autorizacion: a,
	}); !errors.Is(err, ports.ErrUsoCategoriaRPTNoConfiable) || txExtra.confirmada || !txExtra.revertida {
		t.Fatalf("publicación no autorizada expuesta en recibo: %v tx=%+v", err, txExtra)
	}
	wrong := m
	wrong.Consumidor = "personal"
	inicio := &iniciadorUsoRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	gestor, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
	if _, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
		Material: wrong, Solicitud: s, Autorizacion: a,
	}); !errors.Is(err, ports.ErrUsoCategoriaRPTInvalido) || inicio.llamadas != 0 {
		t.Fatalf("consumidor del material eligió pool: %v", err)
	}
	sOtro, aOtro := autorizacionUsoRPTPrueba(t, accionConfirmarUsoRPT, audienciaUsosRPT, m.UsoRef)
	if _, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
		Material: m, Solicitud: sOtro, Autorizacion: aOtro,
	}); !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || inicio.llamadas != 0 {
		t.Fatalf("acción V3 ajena llegó a SQL: %v", err)
	}
	tx := &transaccionLecturaRPTPrueba{huellaMaterial: strings.Repeat("b", 64)}
	gestor, _ = nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
	if _, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
		Material: m, Solicitud: s, Autorizacion: a,
	}); !errors.Is(err, ports.ErrUsoCategoriaRPTDenegado) || tx.fachadas != 0 || tx.confirmada || !tx.revertida {
		t.Fatalf("huella PG divergente consumió V3: %v tx=%+v", err, tx)
	}
}

func TestUsoRPTSQLSTATEYCancelacionNoConfundenAusencia(t *testing.T) {
	m := materialReservaUsoRPTPrueba()
	s, a := autorizacionUsoRPTPrueba(t, accionReservarUsoRPT, audienciaUsosRPT, m.UsoRef)
	for _, caso := range []struct {
		codigo   string
		esperado error
	}{
		{"22023", ports.ErrUsoCategoriaRPTInvalido},
		{"42501", ports.ErrUsoCategoriaRPTDenegado},
		{"23505", ports.ErrUsoCategoriaRPTConflicto},
		{"40001", ports.ErrUsoCategoriaRPTConflicto},
		{"55P03", ports.ErrUsoCategoriaRPTConflicto},
		{"55000", ports.ErrUsoCategoriaRPTConflicto},
		{"08006", ports.ErrUsoCategoriaRPTNoDisponible},
	} {
		t.Run(caso.codigo, func(t *testing.T) {
			tx := &transaccionLecturaRPTPrueba{errConsulta: &pgconn.PgError{Code: caso.codigo}}
			gestor, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: tx}, descriptorRPTPrueba, m.Consumidor)
			if _, err := gestor.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
				Material: m, Solicitud: s, Autorizacion: a,
			}); !errors.Is(err, caso.esperado) || tx.confirmada || !tx.revertida {
				t.Fatalf("SQLSTATE %s se confundió con éxito/ausencia: %v tx=%+v", caso.codigo, err, tx)
			}
		})
	}
	ctx, cancelar := context.WithCancel(t.Context())
	cancelar()
	inicio := &iniciadorUsoRPTPrueba{tx: &transaccionLecturaRPTPrueba{}}
	gestor, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(inicio, descriptorRPTPrueba, m.Consumidor)
	m.Consumidor = "personal" // la cancelación antecede incluso a un material inválido
	if _, err := gestor.ReservarUsoCategoriaRPT(ctx, ports.OrdenReservaUsoCategoriaRPT{
		Material: m, Solicitud: s, Autorizacion: a,
	}); !errors.Is(err, context.Canceled) || inicio.llamadas != 0 {
		t.Fatalf("cancelación abrió transacción: %v", err)
	}
	if err := errorUsoCategoriaRPT(ctx, &pgconn.PgError{Code: "23505"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("SQLSTATE ocultó cancelación: %v", err)
	}
	// Si se pierde la respuesta de COMMIT, el adaptador no presenta el recibo
	// como confirmado; una nueva V3 podrá recuperar la operación por sus refs.
	txCommit := &transaccionUsoRPTPrueba{transaccionLecturaRPTPrueba: &transaccionLecturaRPTPrueba{
		respuesta: reciboUsoRPTPrueba(t, a, materialReservaUsoRPTPrueba(), "reservado",
			materialReservaUsoRPTPrueba().ReservaReciboRef, ""),
	}, errorCommit: errors.New("respuesta perdida")}
	gestorCommit, _ := nuevoGestorUsosCategoriaRPTPostgreSQL(&iniciadorUsoRPTPrueba{tx: txCommit}, descriptorRPTPrueba, materialReservaUsoRPTPrueba().Consumidor)
	resultado, err := gestorCommit.ReservarUsoCategoriaRPT(t.Context(), ports.OrdenReservaUsoCategoriaRPT{
		Material: materialReservaUsoRPTPrueba(), Solicitud: s, Autorizacion: a,
	})
	if !errors.Is(err, ports.ErrUsoCategoriaRPTNoDisponible) || resultado.Encontrado ||
		txCommit.confirmada || !txCommit.revertida {
		t.Fatalf("respuesta perdida de COMMIT presentada como éxito: %+v %v tx=%+v", resultado, err, txCommit)
	}
}
