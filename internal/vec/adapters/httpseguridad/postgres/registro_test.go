package postgres

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

type iniciadorDoble struct {
	transacciones []*transaccionDoble
	llamadas      int
	opciones      []pgx.TxOptions
}

func (i *iniciadorDoble) BeginTx(_ context.Context, opciones pgx.TxOptions) (pgx.Tx, error) {
	i.opciones = append(i.opciones, opciones)
	if i.llamadas >= len(i.transacciones) {
		return nil, errors.New("detalle interno del pool")
	}
	tx := i.transacciones[i.llamadas]
	i.llamadas++
	return tx, nil
}

type transaccionDoble struct {
	pgx.Tx
	filas      [][]any
	consultas  []string
	argumentos [][]any
	errExec    error
	errCommit  error
	commits    int
	rollbacks  int
}

func (t *transaccionDoble) Exec(
	_ context.Context,
	_ string,
	_ ...any,
) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("SELECT 1"), t.errExec
}

func (t *transaccionDoble) QueryRow(_ context.Context, sql string, argumentos ...any) pgx.Row {
	t.consultas = append(t.consultas, sql)
	t.argumentos = append(t.argumentos, append([]any(nil), argumentos...))
	if len(t.filas) == 0 {
		return filaDoble{err: errors.New("detalle interno de consulta")}
	}
	fila := t.filas[0]
	t.filas = t.filas[1:]
	return filaDoble{valores: fila}
}

func (t *transaccionDoble) Commit(context.Context) error {
	t.commits++
	return t.errCommit
}

func (t *transaccionDoble) Rollback(context.Context) error {
	t.rollbacks++
	return nil
}

type filaDoble struct {
	valores []any
	err     error
}

func (f filaDoble) Scan(destinos ...any) error {
	if f.err != nil {
		return f.err
	}
	if len(destinos) != len(f.valores) {
		return errors.New("numero de columnas distinto")
	}
	for indice := range destinos {
		destino := reflect.ValueOf(destinos[indice])
		valor := reflect.ValueOf(f.valores[indice])
		if destino.Kind() != reflect.Pointer || !valor.Type().AssignableTo(destino.Elem().Type()) {
			return errors.New("tipo de columna distinto")
		}
		destino.Elem().Set(valor)
	}
	return nil
}

type seudonimizadorDoble struct {
	resultado SeudonimosAlta
	err       error
	entradas  []IdentificadoresAlta
}

func (s *seudonimizadorDoble) SeudonimizarAlta(
	_ context.Context,
	entrada IdentificadoresAlta,
) (SeudonimosAlta, error) {
	s.entradas = append(s.entradas, entrada)
	return s.resultado, s.err
}

func TestRegistroPostgreSQLAltaAtomicaNoEnviaIDs(t *testing.T) {
	alta := altaValida()
	respuesta := filaAltaValida(alta)
	tx := &transaccionDoble{filas: [][]any{respuesta}}
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
	revalidacion := &iniciadorDoble{}
	seudonimizador := &seudonimizadorDoble{resultado: seudonimosValidos(false)}
	adaptador := nuevoAdaptadorPrueba(t, registro, revalidacion, seudonimizador)

	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil {
		t.Fatal("el alta durable valida no se confirmo")
	}
	if tx.commits != 1 || registro.llamadas != 1 || len(seudonimizador.entradas) != 1 {
		t.Fatal("el alta no uso una unica transaccion y una seudonimizacion")
	}
	comprobarSerializable(t, registro.opciones)
	if len(tx.argumentos) != 1 || len(tx.argumentos[0]) != 20 {
		t.Fatal("contrato SQL de alta inesperado")
	}
	for _, argumento := range tx.argumentos[0] {
		texto, esTexto := argumento.(string)
		if !esTexto {
			continue
		}
		for _, id := range []string{
			alta.EspacioIdentidad,
			alta.AsercionID, alta.SesionID, alta.SujetoID,
			alta.CuentaID, alta.CuentaOrdinariaID,
		} {
			if id != "" && texto == id {
				t.Fatal("un identificador fuente alcanzo PostgreSQL")
			}
		}
	}
	if !strings.Contains(tx.consultas[0], "registrar_sesion_v1") {
		t.Fatal("el alta no uso la funcion cerrada")
	}
}

func TestRegistroPostgreSQLReconciliaCommitAmbiguoPorMismaOperacion(t *testing.T) {
	alta := altaValida()
	fila := filaAltaValida(alta)
	txAlta := &transaccionDoble{
		filas: [][]any{fila}, errCommit: errors.New("resultado de COMMIT desconocido"),
	}
	txReconciliacion := &transaccionDoble{filas: [][]any{fila}}
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{txAlta, txReconciliacion}}
	adaptador := nuevoAdaptadorPrueba(
		t, registro, &iniciadorDoble{},
		&seudonimizadorDoble{resultado: seudonimosValidos(false)},
	)

	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil {
		t.Fatal("un COMMIT confirmado por reconciliacion exacta no se recupero")
	}
	if registro.llamadas != 2 || len(txReconciliacion.argumentos) != 1 ||
		txAlta.argumentos[0][0] != txReconciliacion.argumentos[0][0] {
		t.Fatal("la reconciliacion no uso la operacion exacta de la invocacion")
	}
	if !strings.Contains(txReconciliacion.consultas[0], "reconciliar_registro_sesion_v1") {
		t.Fatal("se reintento el consumo en lugar de reconciliarlo")
	}
}

func TestRegistroPostgreSQLNoAceptaReconciliacionDistinta(t *testing.T) {
	alta := altaValida()
	fila := filaAltaValida(alta)
	filaDistinta := append([]any(nil), fila...)
	filaDistinta[2] = referencia("ses_", "z")
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{
		{filas: [][]any{fila}, errCommit: errors.New("commit incierto")},
		{filas: [][]any{filaDistinta}},
	}}
	adaptador := nuevoAdaptadorPrueba(
		t, registro, &iniciadorDoble{},
		&seudonimizadorDoble{resultado: seudonimosValidos(false)},
	)
	_, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || strings.Contains(err.Error(), "commit") {
		t.Fatal("una reconciliacion manipulada no fallo saneada")
	}
}

func TestRegistroPostgreSQLRevalidaPorPoolSeparado(t *testing.T) {
	consulta := consultaValida(altaValida())
	tx := &transaccionDoble{filas: [][]any{{true}}}
	revalidacion := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
	adaptador := nuevoAdaptadorPrueba(
		t, &iniciadorDoble{}, revalidacion,
		&seudonimizadorDoble{resultado: seudonimosValidos(false)},
	)
	if err := adaptador.ComprobarSesionYCuentaActivas(context.Background(), consulta); err != nil {
		t.Fatal("la sesion activa no se revalido")
	}
	if revalidacion.llamadas != 1 || tx.commits != 1 ||
		!strings.Contains(tx.consultas[0], "revalidar_sesion_y_cuentas_v1") {
		t.Fatal("la revalidacion no uso su capacidad exclusiva")
	}
	comprobarSerializable(t, revalidacion.opciones)
}

func TestRegistroPostgreSQLFallaCerradoAntesDeSQL(t *testing.T) {
	alta := altaValida()
	alta.CuentaID = "MAYUSCULAS"
	registro := &iniciadorDoble{}
	seudonimizador := &seudonimizadorDoble{resultado: seudonimosValidos(false)}
	adaptador := nuevoAdaptadorPrueba(t, registro, &iniciadorDoble{}, seudonimizador)
	_, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || registro.llamadas != 0 ||
		len(seudonimizador.entradas) != 0 {
		t.Fatal("un alta no canonica alcanzo infraestructura")
	}

	alto := altaValida()
	alto.EspacioIdentidad = "https://otro-idp.interno.example"
	_, err = adaptador.ConsumirAsercionYRegistrar(context.Background(), alto)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || registro.llamadas != 0 ||
		len(seudonimizador.entradas) != 0 {
		t.Fatal("un emisor distinto del dominio HMAC pinado alcanzo el conector")
	}

	alto = altaValida()
	seudonimizador.resultado.DominioRef = referencia("idh_", "x")
	_, err = adaptador.ConsumirAsercionYRegistrar(context.Background(), alto)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || registro.llamadas != 0 {
		t.Fatal("un dominio HMAC no fijado alcanzo PostgreSQL")
	}

	seudonimizador.resultado = seudonimosValidos(false)
	seudonimizador.resultado.ClaveVersion = uint64(1 << 63)
	_, err = adaptador.ConsumirAsercionYRegistrar(context.Background(), altaValida())
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || registro.llamadas != 0 {
		t.Fatal("una version HMAC no representable por PostgreSQL alcanzo SQL")
	}
}

func TestConstructorExigePoolsYConectorSeparados(t *testing.T) {
	pool := &iniciadorDoble{}
	seudonimizador := &seudonimizadorDoble{resultado: seudonimosValidos(false)}
	casos := []struct {
		registro, revalidacion iniciadorTransacciones
		seudonimizador         SeudonimizadorAlta
		dominio                string
	}{
		{nil, &iniciadorDoble{}, seudonimizador, dominioHMACPrueba},
		{pool, pool, seudonimizador, dominioHMACPrueba},
		{pool, &iniciadorDoble{}, nil, dominioHMACPrueba},
		{pool, &iniciadorDoble{}, seudonimizador, "idp-libre"},
	}
	for _, caso := range casos {
		if _, err := nuevoRegistroSesionesPostgreSQL(
			caso.registro, caso.revalidacion, caso.seudonimizador,
			espacioIdentidadPrueba, caso.dominio,
			bytes.NewReader(make([]byte, 18)),
		); !errors.Is(err, httpseguridad.ErrRegistroSesionesAusente) {
			t.Fatal("una composicion insegura fue aceptada")
		}
	}
}

func TestSeudonimosRechazanDigestsSinSeparacionDeProposito(t *testing.T) {
	for _, privilegiada := range []bool{false, true} {
		cantidad := 4
		if privilegiada {
			cantidad = 5
		}
		for primera := 0; primera < cantidad; primera++ {
			for segunda := primera + 1; segunda < cantidad; segunda++ {
				seudonimos := seudonimosValidos(privilegiada)
				huellas := []*[32]byte{
					&seudonimos.AsercionIDHMAC, &seudonimos.SesionIDHMAC,
					&seudonimos.SujetoIDHMAC, &seudonimos.CuentaIDHMAC,
					&seudonimos.CuentaOrdinariaIDHMAC,
				}
				*huellas[segunda] = *huellas[primera]
				if seudonimos.valida(
					espacioIdentidadPrueba, dominioHMACPrueba, privilegiada,
				) {
					t.Fatalf(
						"digests de proposito %d y %d iguales aceptados (privilegiada=%t)",
						primera, segunda, privilegiada,
					)
				}
			}
		}
	}
}

func TestReferenciaOperacionUsa144BitsYNoSeRepite(t *testing.T) {
	material := append(bytes.Repeat([]byte{1}, 18), bytes.Repeat([]byte{2}, 18)...)
	lector := bytes.NewReader(material)
	primera, err1 := nuevaReferenciaOperacion(lector)
	segunda, err2 := nuevaReferenciaOperacion(lector)
	if err1 != nil || err2 != nil || primera == segunda ||
		!referenciaTecnicaValida(primera, "opr_") ||
		!referenciaTecnicaValida(segunda, "opr_") ||
		len(strings.TrimPrefix(primera, "opr_")) != 24 {
		t.Fatal("la operacion no conserva 144 bits CSPRNG codificados")
	}
}

const (
	dominioHMACPrueba      = "idh_0123456789abcdefghijklmn"
	espacioIdentidadPrueba = "https://idp.interno.example"
)

func nuevoAdaptadorPrueba(
	t *testing.T,
	registro, revalidacion iniciadorTransacciones,
	seudonimizador SeudonimizadorAlta,
) *RegistroSesionesPostgreSQL {
	t.Helper()
	adaptador, err := nuevoRegistroSesionesPostgreSQL(
		registro, revalidacion, seudonimizador,
		espacioIdentidadPrueba, dominioHMACPrueba,
		bytes.NewReader(bytes.Repeat([]byte{7}, 18*8)),
	)
	if err != nil {
		t.Fatal("crear adaptador de prueba")
	}
	return adaptador
}

func altaValida() httpseguridad.AltaSesionAtomica {
	emitida := time.Date(2026, time.July, 18, 0, 0, 0, 0, time.UTC)
	return httpseguridad.AltaSesionAtomica{
		EspacioIdentidad: espacioIdentidadPrueba,
		AsercionID:       "asercion-id", SesionID: "sesion-id", SujetoID: "sujeto-id",
		CuentaID: "cuenta-id", Superficie: httpseguridad.SuperficieInternaCorporativa,
		MetodoObservado:           domain.AuthMethodKerberos,
		GarantiaObservada:         domain.AuthAssuranceHigh,
		AutenticacionHuellaSHA256: strings.Repeat("a", 64),
		AutenticacionVerificadaEn: emitida.Add(-time.Second), SesionEmitidaEn: emitida,
		AsercionExpiraEn:             emitida.Add(5 * time.Minute),
		PoliticaGarantiaRef:          referencia("pga_", "p"),
		PoliticaGarantiaHuellaSHA256: strings.Repeat("b", 64),
	}
}

func seudonimosValidos(privilegiada bool) SeudonimosAlta {
	resultado := SeudonimosAlta{
		Esquema: EsquemaHMACSHA256V1, EspacioIdentidad: espacioIdentidadPrueba,
		DominioRef: dominioHMACPrueba,
		ClaveID:    "clave-hsm-v1", ClaveVersion: 1,
		AsercionIDHMAC: [32]byte{1}, SesionIDHMAC: [32]byte{2},
		SujetoIDHMAC: [32]byte{3}, CuentaIDHMAC: [32]byte{4},
	}
	if privilegiada {
		resultado.CuentaOrdinariaIDHMAC = [32]byte{5}
	}
	return resultado
}

func filaAltaValida(alta httpseguridad.AltaSesionAtomica) []any {
	revalidada := alta.SesionEmitidaEn.Add(time.Second)
	return []any{
		referencia("aut_", "a"), referencia("ase_", "b"), referencia("ses_", "c"),
		referencia("cse_", "d"), "1", "activa", strings.Repeat("c", 64),
		referencia("cta_", "e"), referencia("cta_", "e"),
		revalidada, alta.AsercionExpiraEn,
	}
}

func consultaValida(alta httpseguridad.AltaSesionAtomica) httpseguridad.ConsultaSesionActiva {
	fila := filaAltaValida(alta)
	return httpseguridad.ConsultaSesionActiva{
		AutenticacionRef: fila[0].(string), AutenticacionHuellaSHA256: alta.AutenticacionHuellaSHA256,
		AsercionRef: fila[1].(string), SesionRef: fila[2].(string),
		CuentaRef: fila[7].(string), CuentaOrdinariaRef: fila[8].(string),
		CuentaPrivilegiada: alta.CuentaPrivilegiada, Superficie: alta.Superficie,
		MetodoObservado: alta.MetodoObservado, GarantiaObservada: alta.GarantiaObservada,
		PoliticaGarantiaRef:          alta.PoliticaGarantiaRef,
		PoliticaGarantiaHuellaSHA256: alta.PoliticaGarantiaHuellaSHA256,
		AutenticacionVerificadaEn:    alta.AutenticacionVerificadaEn,
		SesionEmitidaEn:              alta.SesionEmitidaEn, ControlSesionRef: fila[3].(string),
		ControlSesionRevision: 1, ControlSesionEstado: httpseguridad.EstadoControlSesionActiva,
		ControlSesionHuellaSHA256: fila[6].(string),
		SesionRevalidadaEn:        fila[9].(time.Time), SesionValidaHasta: fila[10].(time.Time),
	}
}

func referencia(prefijo, relleno string) string {
	return prefijo + strings.Repeat(relleno, 24)
}

func comprobarSerializable(t *testing.T, opciones []pgx.TxOptions) {
	t.Helper()
	if len(opciones) == 0 || opciones[0].IsoLevel != pgx.Serializable ||
		opciones[0].AccessMode != pgx.ReadWrite {
		t.Fatal("la operacion no uso una transaccion serializable de escritura")
	}
}
