package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strconv"

	vd "vec-diputacion-granada/internal/vec/domain"
)

// Descriptors are public preparation data. They neither register decisions nor
// install permissions. Their references and hashes come from the common domain.
func describeRBAC(rows []rbacConfig) error {
	if len(rows) == 0 || len(rows) > 32 {
		return errors.New("rbac_size")
	}
	for _, r := range rows {
		if r.Role.Referencia() != r.Control.VersionRolRef || r.Role.Referencia() != r.Assignment.VersionRolRef {
			return errors.New("rbac_reference")
		}
		roleSHA, err := r.Role.HuellaSHA256()
		if err != nil {
			return err
		}
		controlSHA, err := r.Control.HuellaSHA256()
		if err != nil {
			return err
		}
		assignmentSHA, err := r.Assignment.HuellaSHA256()
		if err != nil {
			return err
		}
		role, err := json.Marshal(r.Role)
		if err != nil {
			return err
		}
		control, err := json.Marshal(r.Control)
		if err != nil {
			return err
		}
		assignment, err := json.Marshal(r.Assignment)
		if err != nil {
			return err
		}
		if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"name": r.Name, "role_ref": r.Role.Referencia(), "role": role, "role_sha256": roleSHA, "control": control, "control_sha256": controlSHA, "assignment_ref": r.Assignment.Referencia(), "assignment": assignment, "assignment_sha256": assignmentSHA}); err != nil {
			return err
		}
	}
	return nil
}

func describePlan(operations []operation) error {
	if len(operations) == 0 || len(operations) > 32 {
		return errors.New("plan_size")
	}
	for _, o := range operations {
		b, err := o.Command.RepresentacionCanonica()
		if err != nil {
			return err
		}
		h := sha256.Sum256(b)
		commandSHA := hex.EncodeToString(h[:])
		resource := vd.RecursoAutorizable{Referencia: o.Command.Hecho.Referencia, ModuloID: "meritos", Tipo: "hecho", Ambitos: map[string]string{"persona_ref": o.Command.Hecho.PersonaRef, "version_esperada": strconv.Itoa(o.Command.VersionEsperada), "huella_comando_sha256": commandSHA}}
		resourceSHA, err := resource.HuellaContextoAutorizacionSHA256()
		if err != nil {
			return err
		}
		if err = json.NewEncoder(os.Stdout).Encode(map[string]any{"name": o.Name, "command": b, "command_sha256": commandSHA, "resource": resource, "resource_context_sha256": resourceSHA}); err != nil {
			return err
		}
	}
	return nil
}
