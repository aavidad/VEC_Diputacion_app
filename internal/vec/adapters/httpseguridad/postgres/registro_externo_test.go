package postgres

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
	"vec-diputacion-granada/internal/vec/domain"
)

func altaExternaValida() httpseguridad.AltaSesionAtomica {
	alta := altaValida()
	alta.Superficie = httpseguridad.SuperficieExternaPersonal
	alta.MetodoObservado = domain.AuthMethodCertificate
	return alta
}

func nuevoRegistroExternoPrueba(
	t *testing.T,
	registro, revalidacion iniciadorTransacciones,
) *RegistroSesionesExternoPostgreSQL {
	t.Helper()
	adaptador, err := nuevoRegistroSesionesExternoPostgreSQL(
		registro, revalidacion,
		&seudonimizadorDoble{resultado: seudonimosValidos(false)},
		espacioIdentidadPrueba, dominioHMACPrueba,
		bytes.NewReader(bytes.Repeat([]byte{7}, 18*8)),
	)
	if err != nil {
		t.Fatal(err)
	}
	return adaptador
}

func TestRegistroExternoSoloUsaFuncionesYSuperficieExternas(t *testing.T) {
	alta := altaExternaValida()
	registroTx := &transaccionDoble{filas: [][]any{filaAltaValida(alta)}}
	revalidaTx := &transaccionDoble{filas: [][]any{{true}}}
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{registroTx}}
	revalida := &iniciadorDoble{transacciones: []*transaccionDoble{revalidaTx}}
	adaptador := nuevoRegistroExternoPrueba(t, registro, revalida)
	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil ||
		len(registroTx.consultas) != 1 ||
		!strings.Contains(registroTx.consultas[0], "vec_identidad_externa_v1.registrar_sesion_v1") ||
		strings.Contains(registroTx.consultas[0], "vec_identidad_sesiones_v1.registrar_sesion_v1") ||
		len(registroTx.argumentos[0]) != 20 || registroTx.commits != 1 {
		t.Fatal("el alta externa no uso la frontera exclusiva")
	}
	if err := adaptador.ComprobarSesionYCuentaActivas(
		context.Background(), consultaValida(alta),
	); err != nil || revalidaTx.commits != 1 ||
		len(revalidaTx.consultas) != 1 ||
		!strings.Contains(revalidaTx.consultas[0], "vec_identidad_externa_v1.revalidar_sesion_y_cuentas_v1") {
		t.Fatal("la revalidacion externa no uso la frontera exclusiva")
	}
	comprobarSerializable(t, registro.opciones)
	comprobarSerializable(t, revalida.opciones)
}

func TestRegistroExternoRechazaSuperficieInternaAntesDeSQL(t *testing.T) {
	registro := &iniciadorDoble{}
	revalida := &iniciadorDoble{}
	adaptador := nuevoRegistroExternoPrueba(t, registro, revalida)
	if _, err := adaptador.ConsumirAsercionYRegistrar(
		context.Background(), altaValida(),
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) || registro.llamadas != 0 {
		t.Fatal("una alta interna llego al registro externo")
	}
	if err := adaptador.ComprobarSesionYCuentaActivas(
		context.Background(), consultaValida(altaValida()),
	); !errors.Is(err, httpseguridad.ErrSesionNoValida) || revalida.llamadas != 0 {
		t.Fatal("una consulta interna llego a la revalidacion externa")
	}
}

func TestRegistroExternoRecuperaSoloOperacionOriginal(t *testing.T) {
	alta := altaExternaValida()
	fila := filaAltaValida(alta)
	registroTx := &transaccionDoble{filas: [][]any{fila}, errCommit: errors.New("commit incierto")}
	recuperacionTx := &transaccionDoble{filas: [][]any{fila}}
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{registroTx, recuperacionTx}}
	adaptador := nuevoRegistroExternoPrueba(t, registro, &iniciadorDoble{})
	if _, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta); err != nil ||
		registro.llamadas != 2 || len(recuperacionTx.consultas) != 1 ||
		!strings.Contains(recuperacionTx.consultas[0], "vec_identidad_externa_v1.reconciliar_registro_sesion_v1") ||
		registroTx.argumentos[0][0] != recuperacionTx.argumentos[0][0] ||
		recuperacionTx.commits != 1 {
		t.Fatal("el commit incierto no se recupero por la operacion exacta")
	}
}

func TestRevalidadorActorExternoNoAceptaProyeccionInterna(t *testing.T) {
	solicitud := solicitudRevalidacionActorValida()
	fila := filaRevalidacionActorValida()
	fila[10] = string(domain.SuperficieAutenticacionExternaPersonalV1)
	fila[11] = string(domain.AuthMethodCertificate)
	tx := &transaccionDoble{filas: [][]any{fila}}
	pool := &iniciadorDoble{transacciones: []*transaccionDoble{tx}}
	revalidador, err := nuevoRevalidadorAutenticacionActorExternoPostgreSQL(pool)
	if err != nil {
		t.Fatal(err)
	}
	if resultado, err := revalidador.RevalidarAutenticacionActorV1(
		context.Background(), solicitud,
	); err != nil || resultado.Superficie != domain.SuperficieAutenticacionExternaPersonalV1 ||
		!strings.Contains(tx.consultas[0], "vec_identidad_externa_v1.revalidar_autenticacion_actor_v1") {
		t.Fatal("revalidacion de actor externo inesperada")
	}
	fx := filaRevalidacionActorValida()
	txInterna := &transaccionDoble{filas: [][]any{fx}}
	revalidador, _ = nuevoRevalidadorAutenticacionActorExternoPostgreSQL(
		&iniciadorDoble{transacciones: []*transaccionDoble{txInterna}},
	)
	if _, err := revalidador.RevalidarAutenticacionActorV1(
		context.Background(), solicitud,
	); err == nil {
		t.Fatal("se acepto proyeccion interna")
	}
}
