package bootstrap

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	core "vec-diputacion-granada/internal/vec/domain"
)

func configuracionB2PuraPrueba() archivoIncorporacionV2 {
	motivo := core.ReferenciaEntradaCatalogo{CatalogoID: "motivos_autorizacion", CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "motivo_11111111111111111111111111111111"}
	b2 := &archivoIncorporacionPersonalB2{Protocolo: "personal_b2_v1", OrganismoRef: "organismo:prueba", CatalogoRPTID: "categorias_rpt", ModuloRPTID: "personal", Pools: map[string]string{}, Operaciones: map[string]archivoOperacionIncorporacionB2{}}
	for k := range rolesPoolsIncorporacionB2 {
		b2.Pools[k] = k + ".dsn"
	}
	for _, d := range operacionesIncorporacionB2() {
		b2.Operaciones[d.clave] = archivoOperacionIncorporacionB2{Motivo: motivo, Capacidad: archivoCapacidadIncorporacionV2{File: d.clave + ".cap"}}
	}
	c := archivoIncorporacionV2{Esquema: "vec.contratacion-temporal.incorporacion-servidor.v2", Referencias: ReferenciasCTIncorporacionDesarrollo{PrincipalV3Ref: "principal:prueba", PerfilV3Ref: "perfil:prueba", OrganizacionRef: "ref:" + strings.Repeat("a", 64), UnidadRef: "ref:" + strings.Repeat("b", 64), ActorRef: "ref:" + strings.Repeat("c", 64)}, PersonalB2: b2, Pools: map[string]string{}}
	for k := range rolesPoolsIncorporacionB2Pura {
		c.Pools[k] = k + ".dsn"
	}
	return c
}
func TestIncorporacionB2ConfiguracionPuraSinFuentesDelEjercicio(t *testing.T) {
	dir, e := os.MkdirTemp("/var/tmp", "vec-incorporacion-b2-")
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	if e = os.Chmod(dir, 0700); e != nil {
		t.Fatal(e)
	}
	c := configuracionB2PuraPrueba()
	b, e := json.Marshal(c)
	if e != nil {
		t.Fatal(e)
	}
	ruta := filepath.Join(dir, "incorporacion.json")
	if e = os.WriteFile(ruta, b, 0600); e != nil {
		t.Fatal(e)
	}
	r, raiz, e := leerConfiguracionIncorporacionV2(ruta)
	if e != nil {
		t.Fatal(e)
	}
	defer raiz.Close()
	if r.Planes != "" || r.Personal != "" || r.PersonalB2 == nil || len(r.Pools) != 3 {
		t.Fatal("B2 exigió material del protocolo anterior")
	}
	for _, alterar := range []func(*archivoIncorporacionV2){
		func(c *archivoIncorporacionV2) { c.Personal = "personal.json" },
		func(c *archivoIncorporacionV2) { c.Continuidad = &archivoContinuidadNominal{} },
		func(c *archivoIncorporacionV2) { c.Pools["historia_ct"] = "historia.dsn" },
		func(c *archivoIncorporacionV2) { delete(c.PersonalB2.Operaciones, "ct_origen_confirmar") },
		func(c *archivoIncorporacionV2) {
			c.PersonalB2.Operaciones["conceder"] = c.PersonalB2.Operaciones["ct_plan_preparar"]
		},
		func(c *archivoIncorporacionV2) { c.PersonalB2.Protocolo = "ejercicio_v2" },
	} {
		c := configuracionB2PuraPrueba()
		alterar(&c)
		b, _ := json.Marshal(c)
		if e = os.WriteFile(ruta, b, 0600); e != nil {
			t.Fatal(e)
		}
		if _, r, e := leerConfiguracionIncorporacionV2(ruta); e == nil || r != nil {
			t.Fatal("configuración mezclada o abierta admitida")
		}
	}
}
func TestIncorporacionB2AudienciasNuevasSinColisiones(t *testing.T) {
	nuevos := descriptoresMaterialIncorporacionB2()
	claves := []string{"bolsa_anclaje", "personal_clases", "ct_plan_preparar", "ct_plan_consultar", "ct_origen_confirmar", "bolsa_persona", "ct_vinculo_consultar", "rpt_publicacion", "rpt_reservar", "ct_vinculo_registrar"}
	if len(nuevos) != len(claves) {
		t.Fatalf("audiencias nuevas: %d", len(nuevos))
	}
	operaciones := operacionesIncorporacionB2()
	for i, clave := range claves {
		var audiencia string
		for _, operacion := range operaciones {
			if operacion.clave == clave {
				audiencia = operacion.audiencia
				break
			}
		}
		d := nuevos[i]
		if audiencia == "" || d.Audiencia != audiencia || d.Dominio != "vec.incorporacion-b2."+clave+".capacidad-v3" || d.Prefijo != "clave:capacidad:incorporacion-b2-"+clave+":" || d.ProveedorNominal != "proveedor-material-incorporacion-b2-"+clave {
			t.Fatalf("descriptor %d no corresponde a %s: %+v", i, clave, d)
		}
		if !audienciaConsumoGobiernoPostgreSQLContratacionTemporalDesarrolloEsPropia(d.Audiencia) {
			t.Fatalf("audiencia no publicable: %s", d.Audiencia)
		}
	}
	todos := append(descriptoresPreviosPersonalB2Prueba(), descriptoresMaterialPersonalB2Desarrollo()...)
	todos = append(todos, nuevos...)
	if _, e := nuevoCatalogoMaterialAutorizacionComunDesarrollo(todos); e != nil {
		t.Fatal(e)
	}
}
