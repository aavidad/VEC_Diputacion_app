package main

import (
	"encoding/json"
	"errors"
	"os"
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
