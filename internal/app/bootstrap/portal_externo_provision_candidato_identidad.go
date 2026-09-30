package bootstrap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// IdentidadProvisionCandidatoExterno recibe solo el alias calculado por
// ExportarSeudonimosPortalExterno offline con el material externo separado.
// No contiene identificadores en claro ni claves, y nunca se escribe en el
// almacén común de identidad.
type IdentidadProvisionCandidatoExterno seudonimoCuentaPortalExterno

func (IdentidadProvisionCandidatoExterno) String() string   { return "[ALIAS EXTERNO PRIVADO]" }
func (IdentidadProvisionCandidatoExterno) GoString() string { return "[ALIAS EXTERNO PRIVADO]" }

func (i IdentidadProvisionCandidatoExterno) validarPara(cuenta string) error {
	if i.CuentaRef != cuenta || i.ClaveVersion > 1<<63-1 || !referenciaExterna(cuenta, "cta_") ||
		!huellaPreimagenMiBolsaPortalExterno.MatchString(i.CuentaHMAC) || !huellaPreimagenMiBolsaPortalExterno.MatchString(i.SujetoHMAC) {
		return ErrProvisionCandidatoExterno
	}
	for _, texto := range []string{i.CuentaRef, i.Esquema, i.DominioRef, i.ClaveID} {
		for _, c := range texto {
			if c < 33 || c > 126 {
				return ErrProvisionCandidatoExterno
			}
		}
	}
	b, err := json.Marshal(seudonimosPortalExterno{Version: 1, Cuentas: []seudonimoCuentaPortalExterno{seudonimoCuentaPortalExterno(i)}})
	if err != nil {
		return errors.Join(ErrProvisionCandidatoExterno, err)
	}
	_, err = leerSeudonimosPortalExterno(b)
	if err != nil {
		return errors.Join(ErrProvisionCandidatoExterno, err)
	}
	if i.CuentaHMAC == i.SujetoHMAC {
		return ErrProvisionCandidatoExterno
	}
	return nil
}

// CargarAliasIdentidadCandidatoExterno selecciona una única cuenta del
// resultado privado de ExportarSeudonimosPortalExterno; no registra alias.
func CargarAliasIdentidadCandidatoExterno(ruta, cuenta string) (*IdentidadProvisionCandidatoExterno, error) {
	b, err := LeerMaterialProvisionExterna(ruta, 256<<10)
	if err != nil {
		return nil, ErrProvisionCandidatoExterno
	}
	defer clear(b)
	s, err := leerSeudonimosPortalExterno(b)
	if err != nil {
		return nil, ErrProvisionCandidatoExterno
	}
	var resultado *IdentidadProvisionCandidatoExterno
	for _, c := range s.Cuentas {
		if c.CuentaRef != cuenta {
			continue
		}
		if resultado != nil {
			return nil, ErrProvisionCandidatoExterno
		}
		i := IdentidadProvisionCandidatoExterno(c)
		if i.validarPara(cuenta) != nil {
			return nil, ErrProvisionCandidatoExterno
		}
		resultado = &i
	}
	if resultado == nil {
		return nil, ErrProvisionCandidatoExterno
	}
	return resultado, nil
}

// Estas dos huellas reproducen el canon ID7, que se vuelve a comprobar en
// SQL antes de consumir la aprobación. Los campos textuales son ASCII.
func huellaMaterialIdentidadExterna(i IdentidadProvisionCandidatoExterno) string {
	base := ""
	for _, s := range []string{i.CuentaRef, i.Esquema, i.DominioRef, i.ClaveID} {
		base += strconv.Itoa(len(s)) + ":" + s
	}
	base += strconv.FormatUint(i.ClaveVersion, 10) + ":" + i.CuentaHMAC + ":" + i.SujetoHMAC
	h := sha256.Sum256([]byte(base))
	return hex.EncodeToString(h[:])
}

func huellaAusenciaIdentidadExterna(cuenta string) string {
	h := sha256.Sum256([]byte("vec.identidad.externa.ausente.v1:" + strconv.Itoa(len(cuenta)) + ":" + cuenta))
	return hex.EncodeToString(h[:])
}

func publicarIdentidadCandidatoExterno(ctx context.Context, tx pgx.Tx, p PlanProvisionCandidatoExterno) error {
	if p.identidad == nil || p.identidad.validarPara(p.identidad.CuentaRef) != nil || p.resumen.PreimagenSHA256 != huellaAusenciaIdentidadExterna(p.identidad.CuentaRef) {
		return ErrProvisionCandidatoExterno
	}
	i := *p.identidad
	cuenta, e1 := hex.DecodeString(i.CuentaHMAC)
	sujeto, e2 := hex.DecodeString(i.SujetoHMAC)
	if e1 != nil || e2 != nil {
		return ErrProvisionCandidatoExterno
	}
	defer clear(cuenta)
	defer clear(sujeto)
	material := huellaMaterialIdentidadExterna(i)
	aprobacion := "apr_" + p.resumen.HuellaSHA256
	operacion := "opr_" + p.resumen.HuellaSHA256
	// La aprobación y su consumo pertenecen a una transacción: el rol de
	// escritura del portal externo no puede crear ninguna de estas filas.
	_, err := tx.Exec(ctx, `INSERT INTO vec_identidad_externa_v1.aprobacion_alta(aprobacion_ref,huella_material_sha256,huella_preimagen_sha256,revision_esperada) VALUES($1,$2,$3,0)`, aprobacion, material, p.resumen.PreimagenSHA256)
	if err != nil {
		return ErrProvisionCandidatoExterno
	}
	var ref string
	err = tx.QueryRow(ctx, `SELECT vec_identidad_externa_v1.confirmar_alta_v1($1,$2,$3,$4,$5,$6,$7::bigint,$8,$9,$10,$11,$12)`, operacion, aprobacion, i.CuentaRef, i.Esquema, i.DominioRef, i.ClaveID, strconv.FormatUint(i.ClaveVersion, 10), cuenta, sujeto, material, p.resumen.PreimagenSHA256, int64(0)).Scan(&ref)
	if err != nil || ref != i.CuentaRef {
		return ErrProvisionCandidatoExterno
	}
	return nil
}
