package bootstrap

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProvisionContextoUsuariosRechazaOtraFamiliaYCASAmbiguo(t *testing.T) {
	s := SolicitudContextoUsuariosExterno{Snapshot: SnapshotContextoExterno{Poblacion: "usuarios", Estado: "activo"}, AprobacionRef: "aprobacion:sintetica"}
	for _, caso := range []string{"candidato", "vinculo candidato", "revocado", "CAS sin huella", "CAS inicial con huella"} {
		t.Run(caso, func(t *testing.T) {
			otra := s
			switch caso {
			case "candidato":
				otra.Snapshot.Poblacion = "candidato"
			case "vinculo candidato":
				otra.Snapshot.VinculoCandidato = &VinculoSnapshotCandidatoExterno{}
			case "revocado":
				otra.Snapshot.Estado = "revocado"
			case "CAS sin huella":
				otra.VersionEsperada = 1
			case "CAS inicial con huella":
				otra.HuellaEsperada = strings.Repeat("a", 64)
			}
			b, _ := json.Marshal(otra)
			if _, err := LeerSolicitudContextoUsuariosExterno(b); err == nil {
				t.Fatal("material ajeno o CAS inválido admitido")
			}
		})
	}
	b, _ := json.Marshal(s)
	b = append([]byte(`{"aprobacion_ref":"otra",`), b[1:]...)
	if _, err := LeerSolicitudContextoUsuariosExterno(b); err == nil {
		t.Fatal("propiedad duplicada admitida")
	}
}

func TestProvisionContextoUsuariosHuellaLigaCanonAprobacionYCAS(t *testing.T) {
	s := SolicitudContextoUsuariosExterno{Snapshot: SnapshotContextoExterno{Poblacion: "usuarios", Estado: "activo"}, AprobacionRef: "aprobacion:sintetica"}
	h := strings.Repeat("a", 64)
	r, err := resumenContextoUsuarios(s, h)
	if err != nil || r.HuellaPlan == "" {
		t.Fatal(err)
	}
	for _, caso := range []string{"canon", "aprobacion", "preimagen", "cuenta"} {
		otra, canon := s, h
		switch caso {
		case "canon":
			canon = strings.Repeat("b", 64)
		case "aprobacion":
			otra.AprobacionRef = "aprobacion:otra"
		case "preimagen":
			otra.VersionEsperada = 1
			otra.HuellaEsperada = strings.Repeat("c", 64)
		case "cuenta":
			otra.Snapshot.Cuenta.Referencia = "cta_otra_sintetica"
		}
		cambiada, err := resumenContextoUsuarios(otra, canon)
		if err != nil || cambiada.HuellaPlan == r.HuellaPlan {
			t.Fatalf("material %s fuera de aprobación", caso)
		}
	}
}
