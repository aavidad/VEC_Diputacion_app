package main

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/app/bootstrap"
	pg "vec-diputacion-granada/internal/vec/adapters/administracionperfiles/postgres"
	confianza "vec-diputacion-granada/internal/vec/adapters/seguridad/confianzaatestacion"
	"vec-diputacion-granada/internal/vec/domain"
)

type relojOverlayPrueba struct{ t time.Time }

func (r relojOverlayPrueba) Ahora() time.Time { return r.t }

// El fichero que genera vec-gobierno-usuarios-admin (fase version-bolsa) lo
// acepta tal cual el cargador de VEC_ADMIN_VERSION_BOLSA_CONFIG_FILE, con la
// raíz del firmante de vec-admin y material que se lee de sus ficheros.
func TestOverlayVersionBolsaGeneradoLoAceptaVecAdmin(t *testing.T) {
	gobierno, base, u, runtime, lote, plan := gobiernoRolesConfigPrueba(t)
	meta, err := decodificarMetadatosConfianzaPerfiles(u.ConfianzaJSON)
	if err != nil {
		t.Fatal(err)
	}
	publica, err := base64.StdEncoding.DecodeString(base.Firmante.PublicaEsperadaBase64)
	if err != nil {
		t.Fatal(err)
	}
	conjunto, _ := bootstrap.AudienciasConjuntoCapacidadesAdmin(5)
	ahora := meta.Gobierno.PublicadaEn.Add(10 * time.Minute)
	cfg := bootstrap.ConfiguracionMaterialUsuariosAdmin{DirectorioMaterial: "/srv/privado/material", RutaConfiguracionHMAC: "/srv/privado/hmac.json",
		ArchivoSemillaRaiz: "/srv/privado/semilla", PrefijoEvidencia: "evidencia:firma:admin:usuarios:", ConjuntoVersion: 5,
		Raiz: administracion.MaterialRaizPerfilesV3{ClaveID: base.Firmante.ClaveID, Audiencia: base.Firmante.Audiencia, Version: meta.Raiz.Version,
			Publica: publica, Estado: confianza.EstadoClaveAtestacionAutorizacionV3Activa, ValidaDesde: meta.Raiz.ValidaDesde, ValidaHasta: meta.Raiz.ValidaHasta},
		Gobierno: administracion.GobiernoConfianzaPerfilesV3{Revision: meta.Gobierno.Revision, HuellaSHA256: meta.Gobierno.HuellaSHA256,
			Secuencia: meta.Gobierno.Secuencia, PublicadaEn: meta.Gobierno.PublicadaEn, ExpiraEn: meta.Gobierno.ExpiraEn}}
	// material.json con una clave nueva de 32 bytes por audiencia del conjunto.
	type clave struct {
		Audiencia      string    `json:"audiencia"`
		ClaveID        string    `json:"clave_id"`
		Version        uint64    `json:"version"`
		Revision       uint64    `json:"revision_gobierno"`
		HuellaGobierno string    `json:"huella_gobierno_sha256"`
		Secreto        []byte    `json:"secreto_hmac"`
		HuellaSecreto  string    `json:"huella_secreto_sha256"`
		EmisorID       string    `json:"emisor_id"`
		Desde          time.Time `json:"valida_desde"`
		Hasta          time.Time `json:"valida_hasta"`
	}
	var claves []clave
	secretos := map[string][]byte{}
	for i, a := range conjunto {
		s := make([]byte, 32)
		if _, err := rand.Read(s); err != nil {
			t.Fatal(err)
		}
		h := sha256.Sum256(s)
		secretos[a.Audiencia] = s
		claves = append(claves, clave{a.Audiencia, "clave:capacidad:admin:" + a.Segmento + ":s9:" + hex.EncodeToString(h[:8]), uint64(10 + i), uint64(20 + i),
			strings.Repeat("e", 64), s, hex.EncodeToString(h[:]), a.EmisorID, ahora.Add(-time.Minute), ahora.Add(30 * time.Minute)})
	}
	material, _ := json.Marshal(struct {
		Claves []clave `json:"claves"`
	}{claves})
	datos, _ := json.Marshal(cfg)
	planJSON, _ := json.Marshal(map[string]any{"version": 2, "operacion_ref": "gca_" + strings.Repeat("c", 32), "preparado_en": ahora, "caduca_en": ahora.Add(time.Hour),
		"preimagen_sha256": strings.Repeat("c", 64), "conjunto_version": 5, "clave_ordenes": []int{1, 2, 3, 4, 5, 6, 7, 8},
		"configuracion": map[string]any{"revision": cfg.Gobierno.Revision, "secuencia": cfg.Gobierno.Secuencia, "huella_sha256": cfg.Gobierno.HuellaSHA256,
			"publicada_en": cfg.Gobierno.PublicadaEn, "expira_en": cfg.Gobierno.ExpiraEn}})
	ph, mh := sha256.Sum256(planJSON), sha256.Sum256(material)
	sha := hex.EncodeToString(ph[:])
	acuse, _ := json.Marshal(map[string]any{"estado": "permitido", "codigo": "gobierno_usuarios_registrado",
		"recibo": map[string]any{"auditoria_ref": "aud_v3_gu_" + sha[:32], "secuencia": 7, "huella_sha256": strings.Repeat("a", 64),
			"correlacion_ref": "correlacion_" + strings.Repeat("2", 32), "registrada_en": ahora, "plan_sha256": sha, "preimagen_sha256": strings.Repeat("c", 64),
			"claves_sha256": strings.Repeat("b", 64), "material_sha256": hex.EncodeToString(mh[:]), "configuracion_ref": cfg.Gobierno.Revision, "replay": false},
		"auditoria_intento": map[string]any{"auditoria_ref": "aud_v3_gui_" + strings.Repeat("e", 32), "secuencia": 8, "huella_sha256": strings.Repeat("f", 64),
			"correlacion_ref": "correlacion_" + strings.Repeat("1", 32), "registrada_en": ahora, "solicitud_sha256": sha}})

	salida := filepath.Join(t.TempDir(), "gobierno-dia")
	if err := os.Mkdir(salida, 0700); err != nil {
		t.Fatal(err)
	}
	for nombre, b := range map[string][]byte{bootstrap.ArchivoConfiguracionGobiernoUsuarios: datos, bootstrap.ArchivoMaterialGobiernoUsuarios: material,
		bootstrap.ArchivoPlanGobiernoUsuarios: planJSON, "acuse-aplicar.json": acuse} {
		if err := os.WriteFile(filepath.Join(salida, nombre), b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	raiz, err := os.OpenRoot(salida)
	if err != nil {
		t.Fatal(err)
	}
	defer raiz.Close()
	dir := filepath.Dir(gobierno.PoolGobierno)
	motivo := motivoLotePrueba(u)
	destino := bootstrap.DestinoOverlayVersionBolsaAdmin{DirectorioSalida: salida, PoolGobierno: filepath.Join(dir, "version-bolsa-ejecutor.json"),
		PoolCatalogo: filepath.Join(dir, "version-bolsa-catalogo.json"),
		Motivos:      map[string]domain.ReferenciaEntradaCatalogo{pg.AudienciaVersionarRolBolsaProponer: motivo, pg.AudienciaVersionarRolBolsaAprobar: motivo}}
	if err := bootstrap.EscribirOverlayVersionBolsaAdmin(raiz, destino, "acuse-aplicar.json", relojOverlayPrueba{ahora}); err != nil {
		t.Fatal("generación del overlay:", err)
	}

	ruta := filepath.Join(salida, bootstrap.ArchivoOverlayVersionBolsa)
	c, err := cargarConfiguracionVersionBolsaPrivada(ruta, base, u, runtime, &lote, &plan, nil, &gobierno)
	if err != nil {
		t.Fatal("vec-admin rechaza el overlay generado:", err)
	}
	m, err := decodificarMetadatosConfianzaPerfiles(c.ConfianzaJSON)
	if err != nil {
		t.Fatal(err)
	}
	conf, err := confianzaDesdeMetadata(m)
	if err != nil || len(conf.EntradasCapacidad) != 2 {
		t.Fatal("vec-admin no lee el material de las dos claves B1")
	}
	defer func() {
		for i := range conf.EntradasCapacidad {
			clear(conf.EntradasCapacidad[i].Material)
		}
	}()
	for _, e := range conf.EntradasCapacidad {
		if !bytes.Equal(e.Material, secretos[e.Audiencia]) || e.EmisorID != bootstrap.EmisorVersionBolsaAdmin {
			t.Fatal("material o emisor B1 distinto del publicado", e.Audiencia)
		}
	}
}
