package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"vec-diputacion-granada/internal/modules/contrataciontemporal/cobertura"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

type filaCancelacionDecisionCoberturaO404EPrueba struct {
	ctx      context.Context
	iniciada chan struct{}
}

func (f filaCancelacionDecisionCoberturaO404EPrueba) Scan(...any) error {
	close(f.iniciada)
	<-f.ctx.Done()
	return f.ctx.Err()
}

type transaccionCancelacionDecisionCoberturaO404EPrueba struct {
	*transaccionPreparacionPrueba
	consultaIniciada chan struct{}
	consultas        int
}

func (t *transaccionCancelacionDecisionCoberturaO404EPrueba) QueryRow(
	ctx context.Context,
	_ string,
	_ ...any,
) pgx.Row {
	t.consultas++
	return filaCancelacionDecisionCoberturaO404EPrueba{
		ctx: ctx, iniciada: t.consultaIniciada,
	}
}

func TestEjecutoresDecisionCoberturaO404ECierranSesionAntePanic(t *testing.T) {
	t.Parallel()
	t.Run("lectura_escritura", func(t *testing.T) {
		tx := &transaccionPreparacionPrueba{}
		iniciador := &iniciadorPreparacionPrueba{tx: tx}
		ejecutor, err :=
			nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
		if err != nil {
			t.Fatal(err)
		}
		var retenida *sesionDecisionCoberturaO404E
		var recuperado any
		func() {
			defer func() { recuperado = recover() }()
			_ = ejecutor.EjecutarSesionTCB(
				context.Background(),
				func(p cobertura.SesionTCBOperacionDecisionCobertura) error {
					retenida = p.(*sesionDecisionCoberturaO404E)
					panic("fallo deliberado")
				},
			)
		}()
		if recuperado == nil || retenida == nil ||
			retenida.estado != estadoSesionDecisionCoberturaCerrada ||
			retenida.tx != nil || retenida.ctx != nil ||
			tx.confirmaciones != 0 || tx.reversiones != 1 {
			t.Fatalf(
				"panic no cerró RW: panic=%v sesión=%+v commit=%d rollback=%d",
				recuperado,
				retenida,
				tx.confirmaciones,
				tx.reversiones,
			)
		}
	})
	t.Run("solo_lectura", func(t *testing.T) {
		tx := &transaccionPreparacionPrueba{}
		iniciador := &iniciadorPreparacionPrueba{tx: tx}
		ejecutor, err :=
			nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
				iniciador,
			)
		if err != nil {
			t.Fatal(err)
		}
		var retenida *sesionLecturaPrimariaDecisionCoberturaO404E
		var recuperado any
		func() {
			defer func() { recuperado = recover() }()
			_ = ejecutor.EjecutarLecturaPrimariaTCB(
				context.Background(),
				func(
					p cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura,
				) error {
					retenida = p.(*sesionLecturaPrimariaDecisionCoberturaO404E)
					panic("fallo deliberado")
				},
			)
		}()
		if recuperado == nil || retenida == nil || !retenida.cerrada ||
			retenida.tx != nil || retenida.ctx != nil ||
			tx.confirmaciones != 0 || tx.reversiones != 1 {
			t.Fatalf(
				"panic no cerró RO: panic=%v sesión=%+v commit=%d rollback=%d",
				recuperado,
				retenida,
				tx.confirmaciones,
				tx.reversiones,
			)
		}
	})
}

func TestEjecutorDecisionCoberturaO404ERechazaSesionRetenidaYTardia(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	var retenida *sesionDecisionCoberturaO404E
	disparar := make(chan struct{})
	resultadoTardio := make(chan error, 1)
	err = ejecutor.EjecutarSesionTCB(
		context.Background(),
		func(p cobertura.SesionTCBOperacionDecisionCobertura) error {
			retenida = p.(*sesionDecisionCoberturaO404E)
			go func() {
				<-disparar
				resultadoTardio <- retenida.Abrir(
					cobertura.CabeceraSesionTCBOperacionDecisionCobertura{},
				)
			}()
			retenida.mu.Lock()
			retenida.estado = estadoSesionDecisionCoberturaConsumida
			retenida.confirmada = true
			retenida.mu.Unlock()
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ejecutar: %v", err)
	}
	close(disparar)
	if errTardio := <-resultadoTardio; !errors.Is(
		errTardio,
		errSesionDecisionCoberturaO404EInvalida,
	) {
		t.Fatalf("uso tardío aceptado: %v", errTardio)
	}
	if errRetenido := retenida.Abrir(
		cobertura.CabeceraSesionTCBOperacionDecisionCobertura{},
	); !errors.Is(errRetenido, errSesionDecisionCoberturaO404EInvalida) {
		t.Fatalf("uso retenido aceptado: %v", errRetenido)
	}
}

func TestEjecutorLecturaDecisionCoberturaO404ERechazaSesionRetenidaYTardia(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
			iniciador,
		)
	if err != nil {
		t.Fatal(err)
	}
	var retenida *sesionLecturaPrimariaDecisionCoberturaO404E
	disparar := make(chan struct{})
	resultadoTardio := make(chan error, 1)
	err = ejecutor.EjecutarLecturaPrimariaTCB(
		context.Background(),
		func(
			p cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura,
		) error {
			retenida = p.(*sesionLecturaPrimariaDecisionCoberturaO404E)
			go func() {
				<-disparar
				_, errLectura := retenida.LeerTerminalPrimario(
					context.Background(),
					cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura{},
				)
				resultadoTardio <- errLectura
			}()
			retenida.mu.Lock()
			retenida.usada = true
			retenida.mu.Unlock()
			return nil
		},
	)
	if err != nil {
		t.Fatalf("ejecutar: %v", err)
	}
	close(disparar)
	if errTardio := <-resultadoTardio; !errors.Is(
		errTardio,
		errSesionDecisionCoberturaO404EInvalida,
	) {
		t.Fatalf("uso tardío RO aceptado: %v", errTardio)
	}
	if _, errRetenido := retenida.LeerTerminalPrimario(
		context.Background(),
		cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura{},
	); !errors.Is(errRetenido, errSesionDecisionCoberturaO404EInvalida) {
		t.Fatalf("uso retenido RO aceptado: %v", errRetenido)
	}
}

func TestEjecutorDecisionCoberturaO404ERechazaUsoConcurrente(t *testing.T) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarSesionTCB(
		context.Background(),
		func(p cobertura.SesionTCBOperacionDecisionCobertura) error {
			s := p.(*sesionDecisionCoberturaO404E)
			s.mu.Lock()
			resultado := make(chan error, 1)
			go func() {
				resultado <- s.Abrir(
					cobertura.CabeceraSesionTCBOperacionDecisionCobertura{},
				)
			}()
			errConcurrente := <-resultado
			s.estado = estadoSesionDecisionCoberturaConsumida
			s.confirmada = true
			s.mu.Unlock()
			if !errors.Is(
				errConcurrente,
				errSesionDecisionCoberturaO404EInvalida,
			) {
				return fmt.Errorf("uso concurrente aceptado: %w", errConcurrente)
			}
			return nil
		},
	)
	if !errors.Is(err, errSesionDecisionCoberturaO404EInvalida) ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf(
			"violación concurrente no bloqueó commit: err=%v commit=%d rollback=%d",
			err,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
}

func TestEjecutorLecturaDecisionCoberturaO404ERechazaUsoConcurrente(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
			iniciador,
		)
	if err != nil {
		t.Fatal(err)
	}
	err = ejecutor.EjecutarLecturaPrimariaTCB(
		context.Background(),
		func(
			p cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura,
		) error {
			s := p.(*sesionLecturaPrimariaDecisionCoberturaO404E)
			s.mu.Lock()
			resultado := make(chan error, 1)
			go func() {
				_, errLectura := s.LeerTerminalPrimario(
					context.Background(),
					cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura{},
				)
				resultado <- errLectura
			}()
			errConcurrente := <-resultado
			s.usada = true
			s.mu.Unlock()
			if !errors.Is(
				errConcurrente,
				errSesionDecisionCoberturaO404EInvalida,
			) {
				return fmt.Errorf(
					"uso concurrente RO aceptado: %w",
					errConcurrente,
				)
			}
			return nil
		},
	)
	if !errors.Is(err, errSesionDecisionCoberturaO404EInvalida) ||
		tx.confirmaciones != 0 || tx.reversiones != 1 {
		t.Fatalf(
			"violación concurrente RO no bloqueó commit: "+
				"err=%v commit=%d rollback=%d",
			err,
			tx.confirmaciones,
			tx.reversiones,
		)
	}
}

func TestSesionLecturaDecisionCoberturaO404ECancelacionPropiaNoConsulta(
	t *testing.T,
) {
	t.Parallel()
	tx := &transaccionPreparacionPrueba{}
	sesion := &sesionLecturaPrimariaDecisionCoberturaO404E{
		tx: tx, ctx: context.Background(),
		guardia: nuevaGuardiaCicloDecisionCoberturaO404E(),
	}
	ctx, cancelar := context.WithCancel(context.Background())
	cancelar()
	_, err := sesion.LeerTerminalPrimario(
		ctx,
		cobertura.ConsultaPrimariaSesionTCBOperacionDecisionCobertura{},
	)
	if !errors.Is(err, context.Canceled) || tx.consulta != "" ||
		!sesion.usada {
		t.Fatalf(
			"cancelación propia alcanzó QueryRow: err=%v consulta=%q usada=%t",
			err,
			tx.consulta,
			sesion.usada,
		)
	}
}

func TestEjecutorDecisionCoberturaO404ECancelaOperacionAsincronaEscapada(
	t *testing.T,
) {
	t.Parallel()
	base := &transaccionPreparacionPrueba{}
	tx := &transaccionCancelacionDecisionCoberturaO404EPrueba{
		transaccionPreparacionPrueba: base,
		consultaIniciada:             make(chan struct{}),
	}
	iniciador := &iniciadorPreparacionPrueba{tx: tx}
	ejecutor, err :=
		nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
	if err != nil {
		t.Fatal(err)
	}
	resultadoAsincrono := make(chan error, 1)
	err = ejecutor.EjecutarSesionTCB(
		context.Background(),
		func(p cobertura.SesionTCBOperacionDecisionCobertura) error {
			s := p.(*sesionDecisionCoberturaO404E)
			s.mu.Lock()
			s.estado = estadoSesionDecisionCoberturaLista
			s.rama = cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada
			s.carga = cargaDenegadaMinimaDecisionCoberturaO404EPrueba(t)
			s.mu.Unlock()
			go func() {
				_, errConfirmar := s.Confirmar(context.Background())
				resultadoAsincrono <- errConfirmar
			}()
			<-tx.consultaIniciada
			return nil
		},
	)
	errAsincrono := <-resultadoAsincrono
	if !errors.Is(err, errSesionDecisionCoberturaO404EInvalida) ||
		!errors.Is(errAsincrono, context.Canceled) ||
		tx.consultas != 1 || base.confirmaciones != 0 ||
		base.reversiones != 1 {
		t.Fatalf(
			"escape asíncrono no falló cerrado: ejecutor=%v operación=%v "+
				"consultas=%d commit=%d rollback=%d",
			err,
			errAsincrono,
			tx.consultas,
			base.confirmaciones,
			base.reversiones,
		)
	}
}

func TestSesionDecisionCoberturaO404EErrorConsultaConsumeIntento(
	t *testing.T,
) {
	t.Parallel()
	errConsulta := errors.New("primario no disponible")
	tx := &transaccionPreparacionPrueba{
		fila: filaBytesDecisionCoberturaO404EPrueba{err: errConsulta},
	}
	sesion := nuevaSesionDecisionCoberturaO404E(
		tx,
		context.Background(),
		nuevaGuardiaCicloDecisionCoberturaO404E(),
	)
	sesion.estado = estadoSesionDecisionCoberturaLista
	sesion.rama = cobertura.RamaSesionTCBOperacionDecisionCoberturaDenegada
	sesion.carga = cargaDenegadaMinimaDecisionCoberturaO404EPrueba(t)
	if _, err := sesion.Confirmar(context.Background()); !errors.Is(
		err,
		errConsulta,
	) {
		t.Fatalf("error de consulta ocultado: %v", err)
	}
	consultaPrimera := tx.consulta
	if _, err := sesion.Confirmar(context.Background()); !errors.Is(
		err,
		errSesionDecisionCoberturaO404EInvalida,
	) || tx.consulta != consultaPrimera {
		t.Fatalf("el intento consumido volvió a consultar: %v", err)
	}
}

func TestEjecutoresDecisionCoberturaO404EFallanCerradoAntesDelCallback(
	t *testing.T,
) {
	t.Parallel()
	t.Run("configuracion", func(t *testing.T) {
		errConfig := errors.New("set_config rechazado")
		tx := &transaccionPreparacionPrueba{errConfigurar: errConfig}
		iniciador := &iniciadorPreparacionPrueba{tx: tx}
		ejecutor, _ :=
			nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
		invocado := false
		err := ejecutor.EjecutarSesionTCB(
			context.Background(),
			func(cobertura.SesionTCBOperacionDecisionCobertura) error {
				invocado = true
				return nil
			},
		)
		if !errors.Is(err, errConfig) || invocado ||
			tx.confirmaciones != 0 || tx.reversiones != 1 {
			t.Fatalf("fallo de configuración abierto: %v", err)
		}
	})
	t.Run("cancelacion_previa", func(t *testing.T) {
		iniciador := &iniciadorPreparacionPrueba{
			tx: &transaccionPreparacionPrueba{},
		}
		ejecutor, _ :=
			nuevoEjecutorSesionTCBOperacionDecisionCoberturaPostgreSQL(iniciador)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		err := ejecutor.EjecutarSesionTCB(
			ctx,
			func(cobertura.SesionTCBOperacionDecisionCobertura) error {
				return nil
			},
		)
		if !errors.Is(err, context.Canceled) || iniciador.inicios != 0 {
			t.Fatalf("cancelación previa abrió transacción: %v", err)
		}
	})
	t.Run("cancelacion_previa_solo_lectura", func(t *testing.T) {
		iniciador := &iniciadorPreparacionPrueba{
			tx: &transaccionPreparacionPrueba{},
		}
		ejecutor, _ :=
			nuevoEjecutorLecturaPrimariaTCBOperacionDecisionCoberturaPostgreSQL(
				iniciador,
			)
		ctx, cancelar := context.WithCancel(context.Background())
		cancelar()
		err := ejecutor.EjecutarLecturaPrimariaTCB(
			ctx,
			func(
				cobertura.SesionLecturaPrimariaTCBOperacionDecisionCobertura,
			) error {
				return nil
			},
		)
		if !errors.Is(err, context.Canceled) || iniciador.inicios != 0 {
			t.Fatalf("cancelación previa RO abrió transacción: %v", err)
		}
	})
}

func TestDenegacionDecisionCoberturaO404ELigaAmbitosYAtributos(t *testing.T) {
	t.Parallel()
	recurso := recursoDenegacionDecisionCoberturaO404EPrueba()
	huella, err := recurso.HuellaContextoAutorizacionSHA256()
	if err != nil {
		t.Fatal(err)
	}
	dto := denegacionDecisionCoberturaO404E{
		RecursoRef: recurso.Referencia, RecursoModulo: recurso.ModuloID,
		RecursoTipo:         recurso.Tipo,
		Ambitos:             clonarMapaDecisionCoberturaO404E(recurso.Ambitos),
		Atributos:           clonarMapaDecisionCoberturaO404E(recurso.Atributos),
		RecursoHuellaSHA256: huella,
	}
	if !validarRecursoDenegacionDecisionCoberturaO404E(dto) {
		t.Fatal("el recurso nominal válido fue rechazado")
	}
	dto.Ambitos["organizacion_ref"] = "otra-organizacion"
	if validarRecursoDenegacionDecisionCoberturaO404E(dto) {
		t.Fatal("una mutación de ámbito conservó la huella")
	}
	carga := cargaDenegadaMinimaDecisionCoberturaO404EPrueba(t)
	carga.Denegacion.Atributos["clasificacion"] = "alterado"
	if _, err := codificarCargaConfirmarDecisionCoberturaO404E(
		carga,
	); err == nil {
		t.Fatal("una mutación de atributo llegó a la frontera SQL")
	}
	copia := clonarMapaDecisionCoberturaO404E(recurso.Atributos)
	recurso.Atributos["clasificacion"] = "secreto"
	if copia["clasificacion"] != "interno" {
		t.Fatal("la copia defensiva comparte el mapa de atributos")
	}
}

func TestPreflightDecisionCoberturaO404ELimites(t *testing.T) {
	t.Parallel()
	pruebaMinima := pruebasCanonicasC1DecisionCoberturaO404E{
		Peticion: []byte{1}, Resultado: []byte{2}, Atestacion: []byte{3},
		ConfirmacionTCB: []byte{4}, Catalogo: []byte{5},
		Verificador: []byte{6}, Resumen: []byte{7},
	}
	consumos := make(
		[]consumoC1DecisionCoberturaO404E,
		cobertura.MaximoConsumosC1SesionTCBOperacionDecisionCobertura,
	)
	for i := range consumos {
		consumos[i].Pruebas = pruebaMinima
	}
	carga := cargaConfirmarDecisionCoberturaO404E{
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: []byte{1}, MotivoCanonico: []byte{2},
		},
		ConsumosC1: consumos,
	}
	if _, err := prevalidarCargaDecisionCoberturaO404E(carga); err != nil {
		t.Fatalf("512 consumos mínimos rechazados: %v", err)
	}
	carga.ConsumosC1 = append(carga.ConsumosC1, consumoC1DecisionCoberturaO404E{
		Pruebas: pruebaMinima,
	})
	if _, err := prevalidarCargaDecisionCoberturaO404E(carga); err == nil {
		t.Fatal("513 consumos fueron aceptados")
	}

	frontera := cargaCanonicaEnFronteraDecisionCoberturaO404E(t)
	total, err := prevalidarCargaDecisionCoberturaO404E(frontera)
	if err != nil || total != maximoBytesMaterialCanonicoDecisionCoberturaO404E {
		t.Fatalf("frontera de 8 MiB rechazada: total=%d err=%v", total, err)
	}
	frontera.ConsumosC1[0].Pruebas.Peticion = append(
		frontera.ConsumosC1[0].Pruebas.Peticion,
		1,
	)
	if _, err := prevalidarCargaDecisionCoberturaO404E(frontera); err == nil {
		t.Fatal("material canónico por encima de 8 MiB aceptado")
	}
	pruebaFrontera := pruebasCanonicasC1DecisionCoberturaO404E{
		Peticion: bytes.Repeat(
			[]byte{1},
			maximoBytesPruebaC1DecisionCoberturaO404E,
		),
		Resultado: []byte{1}, Atestacion: []byte{1},
		ConfirmacionTCB: []byte{1}, Catalogo: []byte{1},
		Verificador: []byte{1}, Resumen: []byte{1},
	}
	if _, err := tamanioPruebasC1DecisionCoberturaO404E(
		pruebaFrontera,
	); err != nil {
		t.Fatalf("frontera individual rechazada: %v", err)
	}
	pruebaFrontera.Peticion = append(pruebaFrontera.Peticion, 1)
	if _, err := tamanioPruebasC1DecisionCoberturaO404E(
		pruebaFrontera,
	); err == nil {
		t.Fatal("prueba individual sobredimensionada aceptada")
	}
}

func TestConsumosC1DecisionCoberturaO404EOrdenYDuplicados(t *testing.T) {
	t.Parallel()
	for nombre, caso := range map[string]struct {
		posicion, total, siguiente, esperado uint64
		valido                               bool
	}{
		"primero": {1, 2, 1, 2, true},
		"segundo": {2, 2, 2, 2, true},
		"salto":   {2, 2, 1, 2, false},
		"cero":    {0, 2, 0, 2, false},
		"total":   {1, 3, 1, 2, false},
		"exceso":  {3, 2, 3, 2, false},
	} {
		caso := caso
		t.Run(nombre, func(t *testing.T) {
			if obtenido := posicionConsumoC1DecisionCoberturaO404EValida(
				caso.posicion,
				caso.total,
				caso.siguiente,
				caso.esperado,
			); obtenido != caso.valido {
				t.Fatalf("validez=%t, esperada=%t", obtenido, caso.valido)
			}
		})
	}
	sesion := &sesionDecisionCoberturaO404E{
		peticionesC1: make(map[string]struct{}),
		respuestasC1: make(
			map[claveRespuestaC1DecisionCoberturaO404E]struct{},
		),
	}
	primero := consumoC1DecisionCoberturaO404E{
		OrganizacionRef: "org", PeticionRef: "peticion-1",
		AutoridadRef: "autoridad", Generacion: 1,
		ReciboRespuestaRef: "recibo-1",
	}
	if !sesion.registrarIdentidadConsumoC1(primero) ||
		sesion.registrarIdentidadConsumoC1(primero) {
		t.Fatal("la petición C1 duplicada no se rechazó exactamente una vez")
	}
	otraPeticionMismaRespuesta := primero
	otraPeticionMismaRespuesta.PeticionRef = "peticion-2"
	if sesion.registrarIdentidadConsumoC1(otraPeticionMismaRespuesta) {
		t.Fatal("la identidad de respuesta C1 duplicada fue aceptada")
	}
}

func cargaCanonicaEnFronteraDecisionCoberturaO404E(
	t *testing.T,
) cargaConfirmarDecisionCoberturaO404E {
	t.Helper()
	carga := cargaConfirmarDecisionCoberturaO404E{
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: []byte{1}, MotivoCanonico: []byte{2},
		},
	}
	const consumosNecesarios = 19
	restante := maximoBytesMaterialCanonicoDecisionCoberturaO404E -
		2 - consumosNecesarios*7
	for i := 0; i < consumosNecesarios; i++ {
		p := pruebasCanonicasC1DecisionCoberturaO404E{
			Peticion: []byte{1}, Resultado: []byte{1}, Atestacion: []byte{1},
			ConfirmacionTCB: []byte{1}, Catalogo: []byte{1},
			Verificador: []byte{1}, Resumen: []byte{1},
		}
		destinos := []*[]byte{
			&p.Peticion, &p.Resultado, &p.Atestacion, &p.ConfirmacionTCB,
			&p.Catalogo, &p.Verificador, &p.Resumen,
		}
		for _, destino := range destinos {
			extra := restante
			if extra > maximoBytesPruebaC1DecisionCoberturaO404E-1 {
				extra = maximoBytesPruebaC1DecisionCoberturaO404E - 1
			}
			*destino = bytes.Repeat([]byte{1}, 1+extra)
			restante -= extra
		}
		carga.ConsumosC1 = append(
			carga.ConsumosC1,
			consumoC1DecisionCoberturaO404E{Pruebas: p},
		)
	}
	if restante != 0 {
		t.Fatalf("no se construyó la frontera exacta: restan %d bytes", restante)
	}
	return carga
}

func TestCodificacionDecisionCoberturaO404EIncluyeSietePruebasYResultado(
	t *testing.T,
) {
	t.Parallel()
	carga := cargaConfirmarDecisionCoberturaO404E{
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: []byte{0xaa}, MotivoCanonico: []byte{0xbb},
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{{
			ComprobacionResultado: domain.ResultadoComprobacion("cumple"),
			ComprobacionFuenteRef: "fuente:1",
			ComprobacionReciboRef: "recibo:1",
			ComprobacionEvaluadaEn: time.Date(
				2026, 7, 26, 10, 0, 0, 0, time.UTC,
			),
			Pruebas: pruebasCanonicasC1DecisionCoberturaO404E{
				Peticion: []byte{1}, Resultado: []byte{2},
				Atestacion: []byte{3}, ConfirmacionTCB: []byte{4},
				Catalogo: []byte{5}, Verificador: []byte{6},
				Resumen: []byte{7},
			},
		}},
	}
	contenido, err := codificarCargaDecisionCoberturaO404E(carga)
	if err != nil {
		t.Fatal(err)
	}
	defer borrarBytes(contenido)
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(contenido, &objeto); err != nil {
		t.Fatalf("JSON final inválido: %v\n%s", err, contenido)
	}
	texto := string(contenido)
	for nombre, valor := range map[string]string{
		"peticion_hex": "01", "resultado_hex": "02",
		"atestacion_hex": "03", "confirmacion_tcb_hex": "04",
		"catalogo_hex": "05", "verificador_hex": "06", "resumen_hex": "07",
	} {
		if !strings.Contains(texto, `"`+nombre+`":"`+valor+`"`) {
			t.Fatalf("falta prueba %s: %s", nombre, texto)
		}
	}
	if !strings.Contains(texto, `"comprobacion_resultado":"cumple"`) {
		t.Fatalf("falta el resultado C1 cerrado: %s", texto)
	}
	for _, esperado := range []string{
		`"comprobacion_fuente_ref":"fuente:1"`,
		`"comprobacion_recibo_ref":"recibo:1"`,
		`"comprobacion_evaluada_en":"2026-07-26T10:00:00Z"`,
	} {
		if !strings.Contains(texto, esperado) {
			t.Fatalf("falta coordenada C1 %s: %s", esperado, texto)
		}
	}
}

func TestLimpiezaDecisionCoberturaO404EBorraBuffersYMapas(t *testing.T) {
	t.Parallel()
	decision := []byte{1, 2}
	motivo := []byte{3, 4}
	pruebas := [][]byte{
		{5, 6}, {7, 8}, {9, 10}, {11, 12}, {13, 14}, {15, 16}, {17, 18},
	}
	ambitos := map[string]string{"organizacion_ref": "diputacion"}
	atributos := map[string]string{"clasificacion": "interno"}
	carga := cargaConfirmarDecisionCoberturaO404E{
		DecisionVEC: decisionVECDecisionCoberturaO404E{
			DecisionCanonica: decision, MotivoCanonico: motivo,
		},
		ConsumosC1: []consumoC1DecisionCoberturaO404E{{
			Pruebas: pruebasCanonicasC1DecisionCoberturaO404E{
				Peticion: pruebas[0], Resultado: pruebas[1],
				Atestacion: pruebas[2], ConfirmacionTCB: pruebas[3],
				Catalogo: pruebas[4], Verificador: pruebas[5],
				Resumen: pruebas[6],
			},
		}},
		Denegacion: &denegacionDecisionCoberturaO404E{
			Ambitos: ambitos, Atributos: atributos,
		},
	}
	limpiarCargaConfirmarDecisionCoberturaO404E(&carga)
	limpias := true
	for _, prueba := range pruebas {
		limpias = limpias && bytes.Equal(prueba, []byte{0, 0})
	}
	if !bytes.Equal(decision, []byte{0, 0}) ||
		!bytes.Equal(motivo, []byte{0, 0}) || !limpias ||
		len(ambitos) != 0 || len(atributos) != 0 {
		t.Fatalf(
			"limpieza incompleta: decisión=%v motivo=%v pruebas=%v mapas=%d/%d",
			decision,
			motivo,
			pruebas,
			len(ambitos),
			len(atributos),
		)
	}
}
