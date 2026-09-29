package bootstrap

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
)

type SolicitudContextoUsuariosExterno struct {
	Snapshot        SnapshotContextoExterno `json:"snapshot"`
	VersionEsperada int64                   `json:"version_esperada"`
	HuellaEsperada  string                  `json:"huella_esperada"`
	AprobacionRef   string                  `json:"aprobacion_ref"`
}

type ResumenContextoUsuariosExterno struct {
	Fase           string `json:"fase"`
	HuellaPlan     string `json:"plan_huella_sha256"`
	HuellaContexto string `json:"contexto_huella_sha256"`
	AprobacionRef  string `json:"aprobacion_ref"`
}

func LeerSolicitudContextoUsuariosExterno(b []byte) (SolicitudContextoUsuariosExterno, error) {
	var s SolicitudContextoUsuariosExterno
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if len(b) > 262144 || validarClavesJSONUnicas(b) != nil || d.Decode(&s) != nil || d.Decode(new(any)) != io.EOF ||
		s.Snapshot.Poblacion != "usuarios" || s.Snapshot.Estado != "activo" || s.Snapshot.VinculoCandidato != nil ||
		!textoProvisionUsuariosValido(s.AprobacionRef) || s.VersionEsperada < 0 || s.VersionEsperada >= 1<<31 ||
		(s.VersionEsperada == 0) != (s.HuellaEsperada == "") || (s.HuellaEsperada != "" && !huellaProvisionUsuariosValida(s.HuellaEsperada)) {
		return SolicitudContextoUsuariosExterno{}, errPerfilUsuariosExterno
	}
	return s, nil
}

// OperarContextoUsuariosExterno reutiliza el gobierno CTX15. Sin huella
// aprobada solo consulta preimagen y canon; la confirmación exige el plan
// completo y la referencia de aprobación, y revalida ambos en SERIALIZABLE.
func OperarContextoUsuariosExterno(ctx context.Context, con *pgx.Conn, s SolicitudContextoUsuariosExterno, aprobar, referencia string) (r ResumenContextoUsuariosExterno, err error) {
	if ctx == nil || ctx.Err() != nil || con == nil || (aprobar == "") != (referencia == "") {
		return r, errPerfilUsuariosExterno
	}
	b, e := json.Marshal(s)
	if e != nil {
		return r, errPerfilUsuariosExterno
	}
	if _, e = LeerSolicitudContextoUsuariosExterno(b); e != nil {
		return r, e
	}
	opciones := pgx.TxOptions{IsoLevel: pgx.Serializable}
	if aprobar == "" {
		opciones.AccessMode = pgx.ReadOnly
	}
	tx, e := con.BeginTx(ctx, opciones)
	if e != nil {
		return r, errPerfilUsuariosExterno
	}
	defer func() {
		if e := tx.Rollback(ctx); e != nil && !errors.Is(e, pgx.ErrTxClosed) {
			r = ResumenContextoUsuariosExterno{}
			err = errPerfilUsuariosExterno
		}
	}()
	if PrepararCanalContextoExterno(ctx, tx) != nil {
		return r, errPerfilUsuariosExterno
	}
	h, e := HuellaSnapshotContextoExterno(ctx, tx, s.Snapshot, s.VersionEsperada, s.HuellaEsperada)
	if e != nil {
		return r, errPerfilUsuariosExterno
	}
	r, err = resumenContextoUsuarios(s, h)
	if err != nil {
		return ResumenContextoUsuariosExterno{}, err
	}
	if aprobar == "" {
		return r, nil
	}
	if !huellaProvisionUsuariosValida(aprobar) || aprobar != r.HuellaPlan || referencia != s.AprobacionRef {
		return ResumenContextoUsuariosExterno{}, errPerfilUsuariosExterno
	}
	if PublicarSnapshotContextoExterno(ctx, tx, s.Snapshot, s.VersionEsperada, s.HuellaEsperada, h) != nil || tx.Commit(ctx) != nil {
		return ResumenContextoUsuariosExterno{}, errPerfilUsuariosExterno
	}
	return r, nil
}

func resumenContextoUsuarios(s SolicitudContextoUsuariosExterno, h string) (ResumenContextoUsuariosExterno, error) {
	b, err := json.Marshal(struct {
		Contrato  string                           `json:"contrato"`
		Solicitud SolicitudContextoUsuariosExterno `json:"solicitud"`
		Huella    string                           `json:"contexto_huella_sha256"`
	}{"provision_contexto_usuarios_externo_v1", s, h})
	if err != nil {
		return ResumenContextoUsuariosExterno{}, errPerfilUsuariosExterno
	}
	suma := sha256.Sum256(b)
	return ResumenContextoUsuariosExterno{Fase: "contexto", HuellaPlan: hex.EncodeToString(suma[:]), HuellaContexto: h, AprobacionRef: s.AprobacionRef}, nil
}
