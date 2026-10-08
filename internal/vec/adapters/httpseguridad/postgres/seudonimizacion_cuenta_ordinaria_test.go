package postgres

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

type proveedorAliasPrueba struct {
	resultados []SeudonimosAlta
	errores    []error
	entradas   []IdentificadoresAlta
	cancelar   context.CancelFunc
}

func (p *proveedorAliasPrueba) SeudonimizarAlta(_ context.Context, ids IdentificadoresAlta) (SeudonimosAlta, error) {
	p.entradas = append(p.entradas, ids)
	indice := len(p.entradas) - 1
	if p.cancelar != nil && indice == 1 {
		p.cancelar()
	}
	return p.resultados[indice], p.errores[indice]
}

func idsAliasPrueba(ordinaria bool) IdentificadoresAlta {
	ids := IdentificadoresAlta{
		EspacioIdentidad: espacioIdentidadPrueba,
		AsercionID:       "asercion", SesionID: "sesion", SujetoID: "persona",
		CuentaID: "cuenta-privilegiada",
	}
	if ordinaria {
		ids.CuentaOrdinariaID = "cuenta-ordinaria"
	}
	return ids
}

func proveedorDosCuentasPrueba() *proveedorAliasPrueba {
	primera := seudonimosValidos(true)
	segunda := primera
	segunda.CuentaIDHMAC = [32]byte{6}
	return &proveedorAliasPrueba{
		resultados: []SeudonimosAlta{primera, segunda},
		errores:    []error{nil, nil},
	}
}

func TestAliasOrdinarioUsaSegundoPropositoCuentaSinAlterarOriginal(t *testing.T) {
	proveedor := proveedorDosCuentasPrueba()
	ids := idsAliasPrueba(true)
	original, alias, err := SeudonimizarAltaConAliasCuentaOrdinaria(
		context.Background(), proveedor, ids, espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if err != nil || !reflect.DeepEqual(original, proveedor.resultados[0]) ||
		!reflect.DeepEqual(alias, proveedor.resultados[1].CuentaIDHMAC[:]) ||
		reflect.DeepEqual(alias, original.CuentaOrdinariaIDHMAC[:]) {
		t.Fatal("el alias ordinario no procede de CuentaIDHMAC de la segunda consulta")
	}
	if len(proveedor.entradas) != 2 || proveedor.entradas[0] != ids {
		t.Fatal("no se conservaron los identificadores originales")
	}
	esperada := ids
	esperada.CuentaID = ids.CuentaOrdinariaID
	if proveedor.entradas[1] != esperada {
		t.Fatal("la segunda consulta cambió algo distinto de CuentaID")
	}
}

func TestAliasSinCuentaOrdinariaUsaUnaLlamadaYSQLNulo(t *testing.T) {
	proveedor := &proveedorAliasPrueba{
		resultados: []SeudonimosAlta{seudonimosValidos(false)}, errores: []error{nil},
	}
	original, alias, err := SeudonimizarAltaConAliasCuentaOrdinaria(
		context.Background(), proveedor, idsAliasPrueba(false), espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if err != nil || len(proveedor.entradas) != 1 || alias != nil {
		t.Fatal("una cuenta sin ordinaria cambió la cardinalidad o creó un alias")
	}
	argumentos := argumentosAlta("opr_prueba", original, alias, altaValida())
	if argumentos[9] != nil {
		t.Fatal("la cuenta ordinaria ausente no llegó como NULL a SQL")
	}
}

func TestAliasOrdinarioFallaCerradoSiSegundaConsultaDiverge(t *testing.T) {
	casos := map[string]func(*SeudonimosAlta){
		"rotacion":      func(s *SeudonimosAlta) { s.ClaveVersion++ },
		"sujeto":        func(s *SeudonimosAlta) { s.SujetoIDHMAC = [32]byte{8} },
		"asercion":      func(s *SeudonimosAlta) { s.AsercionIDHMAC = [32]byte{8} },
		"sesion":        func(s *SeudonimosAlta) { s.SesionIDHMAC = [32]byte{8} },
		"quinta_huella": func(s *SeudonimosAlta) { s.CuentaOrdinariaIDHMAC = [32]byte{8} },
		"misma_cuenta":  func(s *SeudonimosAlta) { s.CuentaIDHMAC = [32]byte{4} },
	}
	for nombre, mutar := range casos {
		t.Run(nombre, func(t *testing.T) {
			proveedor := proveedorDosCuentasPrueba()
			mutar(&proveedor.resultados[1])
			original, alias, err := SeudonimizarAltaConAliasCuentaOrdinaria(
				context.Background(), proveedor, idsAliasPrueba(true), espacioIdentidadPrueba, dominioHMACPrueba,
			)
			if !errors.Is(err, httpseguridad.ErrSesionNoValida) || alias != nil || original != (SeudonimosAlta{}) {
				t.Fatal("una segunda respuesta divergente entregó material parcial")
			}
		})
	}
}

func TestAliasOrdinarioNiegaFalloYCancelacionDeSegundaConsulta(t *testing.T) {
	proveedor := proveedorDosCuentasPrueba()
	proveedor.errores[1] = errors.New("token retirado")
	original, alias, err := SeudonimizarAltaConAliasCuentaOrdinaria(
		context.Background(), proveedor, idsAliasPrueba(true), espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) || alias != nil || original != (SeudonimosAlta{}) {
		t.Fatal("el fallo del token entregó material parcial")
	}
	ctx, cancel := context.WithCancel(context.Background())
	proveedor = proveedorDosCuentasPrueba()
	proveedor.cancelar = cancel
	original, alias, err = SeudonimizarAltaConAliasCuentaOrdinaria(
		ctx, proveedor, idsAliasPrueba(true), espacioIdentidadPrueba, dominioHMACPrueba,
	)
	if !errors.Is(err, context.Canceled) || alias != nil || original != (SeudonimosAlta{}) {
		t.Fatal("la cancelación durante la segunda consulta entregó material parcial")
	}
}

func TestRegistroPrivilegiadoNiegaAntesDeSQLSiFallaSegundoHMAC(t *testing.T) {
	alta := altaValida()
	alta.CuentaPrivilegiada = true
	alta.CuentaOrdinariaID = "cuenta-ordinaria"
	alta.Superficie = httpseguridad.SuperficieAdministracionPrivilegiada
	proveedor := proveedorDosCuentasPrueba()
	proveedor.errores[1] = errors.New("token retirado")
	registro := &iniciadorDoble{}
	adaptador := nuevoAdaptadorPrueba(t, registro, &iniciadorDoble{}, proveedor)
	_, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if !errors.Is(err, httpseguridad.ErrSesionNoValida) ||
		len(proveedor.entradas) != 2 || registro.llamadas != 0 {
		t.Fatal("el fallo del segundo HMAC abrió una transacción SQL")
	}
}

func TestRegistroPrivilegiadoReutilizaAliasEnReconciliacion(t *testing.T) {
	alta := altaValida()
	alta.CuentaPrivilegiada = true
	alta.CuentaOrdinariaID = "cuenta-ordinaria"
	alta.Superficie = httpseguridad.SuperficieAdministracionPrivilegiada
	fila := filaAltaValida(alta)
	fila[8] = referencia("cta_", "f")
	txAlta := &transaccionDoble{filas: [][]any{fila}, errCommit: errors.New("commit desconocido")}
	txRecuperacion := &transaccionDoble{filas: [][]any{fila}}
	registro := &iniciadorDoble{transacciones: []*transaccionDoble{txAlta, txRecuperacion}}
	proveedor := proveedorDosCuentasPrueba()
	adaptador := nuevoAdaptadorPrueba(t, registro, &iniciadorDoble{}, proveedor)
	confirmacion, err := adaptador.ConsumirAsercionYRegistrar(context.Background(), alta)
	if err != nil || confirmacion.ValidarPara(alta) != nil {
		t.Fatal("el registro privilegiado no recuperó el resultado ambiguo")
	}
	if len(proveedor.entradas) != 2 || registro.llamadas != 2 ||
		len(txAlta.argumentos) != 1 || len(txRecuperacion.argumentos) != 1 ||
		!reflect.DeepEqual(txAlta.argumentos[0], txRecuperacion.argumentos[0]) {
		t.Fatal("la reconciliación recalculó el alias o cambió argumentos")
	}
	alias, ok := txAlta.argumentos[0][9].([]byte)
	if !ok || !bytes.Equal(alias, proveedor.resultados[1].CuentaIDHMAC[:]) ||
		bytes.Equal(alias, proveedor.resultados[0].CuentaOrdinariaIDHMAC[:]) {
		t.Fatal("SQL recibió la finalidad equivocada para la cuenta ordinaria")
	}
}
