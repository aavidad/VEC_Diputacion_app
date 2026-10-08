package postgres

import (
	"context"

	"vec-diputacion-granada/internal/vec/adapters/httpseguridad"
)

// SeudonimizarAltaConAliasCuentaOrdinaria conserva el resultado original del
// proveedor y obtiene por separado la huella de la cuenta ordinaria con el
// propósito "cuenta". IS15 provisiona esa cuenta con ese mismo propósito.
// Ningún resultado parcial puede llegar al registro SQL.
func SeudonimizarAltaConAliasCuentaOrdinaria(
	ctx context.Context,
	proveedor SeudonimizadorAlta,
	ids IdentificadoresAlta,
	espacioIdentidad, dominioHMACRef string,
) (SeudonimosAlta, []byte, error) {
	fallo := func() (SeudonimosAlta, []byte, error) {
		return SeudonimosAlta{}, nil, errorSesionSaneado(ctx)
	}
	if valorNulo(ctx) || valorNulo(proveedor) ||
		!espacioIdentidadValido(espacioIdentidad) ||
		!referenciaTecnicaValida(dominioHMACRef, "idh_") ||
		ids.EspacioIdentidad != espacioIdentidad ||
		ids.CuentaID == "" ||
		ids.CuentaOrdinariaID != "" && ids.CuentaOrdinariaID == ids.CuentaID {
		return SeudonimosAlta{}, nil, httpseguridad.ErrSesionNoValida
	}
	if ctx.Err() != nil {
		return fallo()
	}
	original, err := proveedor.SeudonimizarAlta(ctx, ids)
	conOrdinaria := ids.CuentaOrdinariaID != ""
	if err != nil || ctx.Err() != nil ||
		!original.valida(espacioIdentidad, dominioHMACRef, conOrdinaria) {
		return fallo()
	}
	if !conOrdinaria {
		return original, nil, nil
	}
	consultaOrdinaria := ids
	consultaOrdinaria.CuentaID = ids.CuentaOrdinariaID
	segunda, err := proveedor.SeudonimizarAlta(ctx, consultaOrdinaria)
	if err != nil || ctx.Err() != nil ||
		!segunda.valida(espacioIdentidad, dominioHMACRef, true) ||
		segunda.Esquema != original.Esquema ||
		segunda.EspacioIdentidad != original.EspacioIdentidad ||
		segunda.DominioRef != original.DominioRef ||
		segunda.ClaveID != original.ClaveID ||
		segunda.ClaveVersion != original.ClaveVersion ||
		segunda.SujetoIDHMAC != original.SujetoIDHMAC ||
		segunda.AsercionIDHMAC != original.AsercionIDHMAC ||
		segunda.SesionIDHMAC != original.SesionIDHMAC ||
		segunda.CuentaOrdinariaIDHMAC != original.CuentaOrdinariaIDHMAC ||
		segunda.CuentaIDHMAC == original.CuentaIDHMAC {
		return fallo()
	}
	alias := make([]byte, len(segunda.CuentaIDHMAC))
	copy(alias, segunda.CuentaIDHMAC[:])
	return original, alias, nil
}
