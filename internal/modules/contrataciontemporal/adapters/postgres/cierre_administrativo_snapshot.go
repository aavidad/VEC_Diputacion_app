package postgres

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"vec-diputacion-granada/internal/modules/contrataciontemporal/domain"
)

// PrepararSnapshotCierreAdministrativo valida la cadena completa con ambas
// publicaciones. No altera el codec V1 ni sustituye la raiz fundacional.
func PrepararSnapshotCierreAdministrativo(original, sucesora domain.DefinicionSeguimiento, seguimiento domain.Seguimiento) (SnapshotSeguimientoPersistido, error) {
	canon, err := domain.SerializarEstadoSeguimientoConContinuacionCanonico(original, sucesora, seguimiento.Estado())
	if err != nil || len(canon) == 0 || len(canon) > maximoBytesSnapshotSeguimientoCanon {
		return SnapshotSeguimientoPersistido{}, ErrSeguimientoPersistidoNoConfiable
	}
	b, err := json.Marshal(seguimiento.Estado())
	if err != nil || len(b) > maximoBytesSnapshotSeguimientoJSON {
		return SnapshotSeguimientoPersistido{}, ErrSeguimientoPersistidoNoConfiable
	}
	h := sha256.Sum256(canon)
	return SnapshotSeguimientoPersistido{EstadoJSON: b, EstadoCanonico: canon, HuellaCanonicaSHA256: hex.EncodeToString(h[:])}, nil
}

func restaurarSnapshotCierreAdministrativo(original, sucesora domain.DefinicionSeguimiento, estado domain.EstadoPersistidoSeguimiento, canonHex, huella string) (domain.Seguimiento, error) {
	canon, err := hex.DecodeString(canonHex)
	if err != nil || len(canon) == 0 || len(canon) > maximoBytesSnapshotSeguimientoCanon {
		return domain.Seguimiento{}, ErrSeguimientoPersistidoNoConfiable
	}
	suma := sha256.Sum256(canon)
	if hex.EncodeToString(suma[:]) != huella {
		return domain.Seguimiento{}, ErrSeguimientoPersistidoNoConfiable
	}
	s, err := domain.RehidratarSeguimientoConContinuacion(original, sucesora, estado)
	if err != nil {
		return domain.Seguimiento{}, ErrSeguimientoPersistidoNoConfiable
	}
	esperado, err := domain.SerializarEstadoSeguimientoConContinuacionCanonico(original, sucesora, s.Estado())
	if err != nil || !bytes.Equal(esperado, canon) {
		return domain.Seguimiento{}, ErrSeguimientoPersistidoNoConfiable
	}
	return s, nil
}
