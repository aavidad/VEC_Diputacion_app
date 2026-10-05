package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vec-diputacion-granada/internal/app/administracion"
	"vec-diputacion-granada/internal/vec/domain"
)

func usuariosMetadatosPrueba(t *testing.T, base configuracionPerfilesPrivada) configuracionUsuariosMetadatosPrivada {
	t.Helper()
	dir := t.TempDir()
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	var confianza metadatosConfianzaPerfilesPrivados
	if err := json.Unmarshal(base.ConfianzaJSON, &confianza); err != nil {
		t.Fatal(err)
	}
	e := confianza.EntradasCapacidad[0]
	e.Audiencia = administracion.AudienciaUsuariosListarV3
	e.MaterialArchivo = filepath.Join(dir, "listar.bin")
	f := e
	f.Audiencia = administracion.AudienciaUsuariosConsultarV3
	f.MaterialArchivo = filepath.Join(dir, "consultar.bin")
	confianza.EntradasCapacidad = []capacidadConfianzaPerfilesPrivada{e, f}
	b, err := json.Marshal(confianza)
	if err != nil {
		t.Fatal(err)
	}
	motivo := domain.ReferenciaEntradaCatalogo{CatalogoID: base.CatalogoMotivosID, CatalogoVersion: 1, CatalogoHuellaSHA256: strings.Repeat("a", 64), EntradaClave: "lectura"}
	c := configuracionUsuariosMetadatosPrivada{Modo: modoUsuariosMetadatos, PoolLector: filepath.Join(dir, "lector.json"), PoolIntentos: filepath.Join(dir, "intentos.json"), PoolSelector: filepath.Join(dir, "selector.json"), PoolFronteraTecnica: filepath.Join(dir, "frontera_tecnica.json"), ConfianzaJSON: b,
		OrganizacionRef: "org_bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", UnidadRef: "unidad_admin_sintetica", Proceso: "vec_admin", Canal: "administracion_privilegiada",
		MotivosUsuarios: map[string]domain.ReferenciaEntradaCatalogo{administracion.AudienciaUsuariosListarV3: motivo, administracion.AudienciaUsuariosConsultarV3: motivo},
		MotivoDenegado:  motivo, MotivoError: motivo, PlazoAuditoriaMS: 500, Destinos: map[string]destinoUsuariosPrivado{}}
	for clave := range clavesDestinoUsuarios {
		c.Destinos[clave] = destinoUsuariosPrivado{Accion: "administracion.perfiles.frontera." + clave, RecursoRef: "administracion:perfiles:" + clave, FinalidadRef: "gestion_perfiles", TipoRecurso: "fijo"}
	}
	c.Destinos["buscar_personas"] = destinoUsuariosPrivado{Accion: "administracion.usuarios.listar", RecursoRef: referenciaConjuntoUsuarios(c.OrganizacionRef, c.UnidadRef), FinalidadRef: "gestion_usuarios", TipoRecurso: "conjunto"}
	c.Destinos["consultar_persona"] = destinoUsuariosPrivado{Accion: "administracion.usuarios.consultar", FinalidadRef: "gestion_usuarios", TipoRecurso: "persona"}
	return c
}

func TestOverlayUsuariosPrivadoCerradoYLegadoIntacto(t *testing.T) {
	base := configuracionPrivadaPrueba(t)
	u := usuariosMetadatosPrueba(t, base)
	if err := validarConfiguracionUsuariosMetadatosPrivada(u, base); err != nil {
		t.Fatal(err)
	}
	if got := referenciaConjuntoUsuarios(u.OrganizacionRef, u.UnidadRef); got != "conjunto_admin:dfa8fa3eef2981ce04ad7dcccbee704d" {
		t.Fatalf("conjunto AUT43 divergente: %s", got)
	}
	b, err := json.Marshal(u)
	if err != nil {
		t.Fatal(err)
	}
	var decodificada configuracionUsuariosMetadatosPrivada
	if err := decodificarConfiguracionPrivada(b, &decodificada); err != nil {
		t.Fatal("JSON overlay cerrado rechazado", err)
	}
	if err := validarConfiguracionUsuariosMetadatosPrivada(decodificada, base); err != nil {
		t.Fatal("overlay rehidratado inválido", err)
	}
	ruta := archivoPrivadoPrueba(t, string(b))
	if leido, err := leerArchivoPrivadoPerfiles(ruta); err != nil {
		t.Fatal("lectura privada overlay", len(b), err)
	} else if len(leido) != len(b) {
		t.Fatal("bytes overlay alterados")
	}
	leida, err := cargarConfiguracionUsuariosMetadatosPrivada(ruta, base)
	if err != nil || leida.Modo != modoUsuariosMetadatos || leida.Destinos["buscar_personas"].RecursoRef != referenciaConjuntoUsuarios(u.OrganizacionRef, u.UnidadRef) {
		t.Fatalf("overlay no conservado: %v", err)
	}
	if err := validarConfiguracionPerfilesPrivada(base); err != nil {
		t.Fatal("configuración heredada alterada", err)
	}
}

func TestOverlayUsuariosFallaCerradoSinScopeDosAudienciasYPools(t *testing.T) {
	base := configuracionPrivadaPrueba(t)
	for _, caso := range []struct {
		nombre string
		muta   func(*configuracionUsuariosMetadatosPrivada)
	}{
		{"modo", func(c *configuracionUsuariosMetadatosPrivada) { c.Modo = "legacy" }},
		{"scope", func(c *configuracionUsuariosMetadatosPrivada) { c.OrganizacionRef = "otro" }},
		{"conjunto", func(c *configuracionUsuariosMetadatosPrivada) {
			d := c.Destinos["buscar_personas"]
			d.RecursoRef = "conjunto_admin:" + strings.Repeat("0", 32)
			c.Destinos["buscar_personas"] = d
		}},
		{"pool_repetido", func(c *configuracionUsuariosMetadatosPrivada) { c.PoolLector = base.Pools.FuenteAutorizacion }},
		{"motivo_ausente", func(c *configuracionUsuariosMetadatosPrivada) {
			delete(c.MotivosUsuarios, administracion.AudienciaUsuariosListarV3)
		}},
		{"audiencia_prestada", func(c *configuracionUsuariosMetadatosPrivada) {
			var x metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(c.ConfianzaJSON, &x)
			x.EntradasCapacidad[1].Audiencia = "vec.admin.otra.v1"
			c.ConfianzaJSON, _ = json.Marshal(x)
		}},
		{"tercera_audiencia", func(c *configuracionUsuariosMetadatosPrivada) {
			var x metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(c.ConfianzaJSON, &x)
			e := x.EntradasCapacidad[0]
			e.Audiencia = "vec.admin.tercera.v1"
			e.MaterialArchivo = c.PoolLector + ".hmac"
			x.EntradasCapacidad = append(x.EntradasCapacidad, e)
			c.ConfianzaJSON, _ = json.Marshal(x)
		}},
		{"clave_inline", func(c *configuracionUsuariosMetadatosPrivada) {
			c.ConfianzaJSON = json.RawMessage(`{"material_base64":"privado"}`)
		}},
		{"clave_prestada", func(c *configuracionUsuariosMetadatosPrivada) {
			var x, anterior metadatosConfianzaPerfilesPrivados
			_ = json.Unmarshal(c.ConfianzaJSON, &x)
			_ = json.Unmarshal(base.ConfianzaJSON, &anterior)
			x.EntradasCapacidad[0].MaterialArchivo = anterior.EntradasCapacidad[0].MaterialArchivo
			c.ConfianzaJSON, _ = json.Marshal(x)
		}},
	} {
		t.Run(caso.nombre, func(t *testing.T) {
			c := usuariosMetadatosPrueba(t, base)
			caso.muta(&c)
			if err := validarConfiguracionUsuariosMetadatosPrivada(c, base); !errors.Is(err, errConfiguracionPrivadaPerfiles) {
				t.Fatalf("aceptó overlay %s", caso.nombre)
			}
		})
	}
	baseOK := usuariosMetadatosPrueba(t, base)
	b, err := json.Marshal(baseOK)
	if err != nil {
		t.Fatal(err)
	}
	mutado := strings.Replace(string(b), `"modo":"metadatos_v1"`, `"modo":"metadatos_v1","extra":true`, 1)
	if _, err := cargarConfiguracionUsuariosMetadatosPrivada(archivoPrivadoPrueba(t, mutado), base); err == nil {
		t.Fatal("JSON abierto aceptado")
	}
}
