package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"vec-diputacion-granada/internal/vec/domain"
	"vec-diputacion-granada/internal/vec/ports"
)

type iniciadorContactoPrueba struct{ llamadas int }

func (i *iniciadorContactoPrueba) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	i.llamadas++
	return nil, errors.New("no debe abrir")
}

type protectorContactoPrueba struct{ descifrados int }

func (p *protectorContactoPrueba) CifrarContactoUsuario(context.Context, string, uint64, []byte) (ports.SobreContactoUsuario, error) {
	return ports.SobreContactoUsuario{}, errors.New("no")
}
func (p *protectorContactoPrueba) ConContactoUsuarioDescifrado(context.Context, string, ports.SobreContactoUsuario, func([]byte) error) error {
	p.descifrados++
	return nil
}

func TestContactoUsuarioPostgreSQLRechazaDependenciasAusentes(t *testing.T) {
	if r, err := nuevoRegistroContactoUsuarioPostgreSQL(nil); err == nil || r != nil {
		t.Fatal("el escritor sin pool debe denegarse")
	}
	if r, err := nuevoResolutorContactoUsuarioPostgreSQL(nil, nil); err == nil || r != nil {
		t.Fatal("el lector sin pool/protector debe denegarse")
	}
}

func TestArgumentosContactoMantieneOrdenMaterialV3DeCatorcePiezas(t *testing.T) {
	negocio, recurso, auditoria := []byte("negocio"), []byte("recurso"), []byte("auditoria")
	a := argumentosContacto("vec.contacto_usuario.consultar", ports.ExportacionMaterialConsumoAutorizacionAtestadaV3{}, negocio, recurso, auditoria)
	if len(a) != 14 || a[0] != "vec.contacto_usuario.consultar" || a[5] != "0" || a[6] != "0" || a[11] == nil || string(a[11].([]byte)) != "negocio" || string(a[12].([]byte)) != "recurso" || string(a[13].([]byte)) != "auditoria" {
		t.Fatal("el contrato SQL exige acción, diez piezas V3 y negocio/recurso/auditoría en ese orden")
	}
}

func TestLecturaInvalidaNoAbreTransaccionNiDescifraNiActivaCallback(t *testing.T) {
	p := &protectorContactoPrueba{}
	i := &iniciadorContactoPrueba{}
	r, err := nuevoResolutorContactoUsuarioPostgreSQL(i, p)
	if err != nil {
		t.Fatal(err)
	}
	activo := 0
	err = r.ConContactoUsuario(context.Background(), ports.SolicitudAccesoContactoUsuario{}, func(domain.ContactoUsuario) error { activo++; return nil })
	if err == nil || i.llamadas != 0 || p.descifrados != 0 || activo != 0 {
		t.Fatal("material o evidencia inválidos no pueden alcanzar descifrado ni callback")
	}
}

func TestContactoRecursoAuditoriaSoloSerializaContextoCanonico(t *testing.T) {
	recurso := domain.RecursoAutorizable{
		Referencia: "per_abcdefghijklmnopqrstuv",
		ModuloID:   "vec.module.usuarios", Tipo: "contacto_usuario",
		Ambitos:   map[string]string{"sujeto": "per_abcdefghijklmnopqrstuv"},
		Atributos: map[string]string{"material_sha256": "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
	}
	raw, _, err := contactoRecursoAuditoria(recurso, domain.AuditEntry{})
	if err != nil {
		t.Fatalf("serializar recurso: %v", err)
	}
	var objeto map[string]json.RawMessage
	if err := json.Unmarshal(raw, &objeto); err != nil {
		t.Fatalf("recurso JSON: %v", err)
	}
	if len(objeto) != 2 || objeto["ambitos"] == nil || objeto["atributos"] == nil || objeto["referencia"] != nil {
		t.Fatal("el material para SQL debe ser exactamente ámbitos y atributos")
	}
}
