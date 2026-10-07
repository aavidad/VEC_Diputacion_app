package bootstrap

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type relojGobiernoUsuariosEnsayo struct{}

func (relojGobiernoUsuariosEnsayo) Ahora() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// Opt-in exclusivo del clon sintético privado. El test normal no abre PG ni
// genera material; preparación y aplicación son fases separadas.
func TestGobiernoUsuariosPostgreSQLPrivado(t *testing.T) {
	ruta := os.Getenv("VEC_GOBIERNO_USUARIOS_ENSAYO_CONFIG")
	if ruta == "" {
		t.Skip("ensayo_privado_no_configurado")
	}
	b, err := leerFicheroMaterialSeguro(ruta, 16384)
	if err != nil {
		t.Fatal("ensayo_config_invalida")
	}
	defer borrarBytes(b)
	var f configuracionEnsayoGobiernoUsuarios
	if decodificarGobiernoUsuarios(b, &f) != nil || validarConfiguracionEnsayoGobiernoUsuarios(f) != nil {
		t.Fatal("ensayo_config_invalida")
	}
	raizSalida, err := AbrirRaizPrivadaDenominacionPersona(filepath.Join(f.Salida, ArchivoConfiguracionGobiernoUsuarios))
	if err != nil {
		t.Fatal("ensayo_salida_invalida")
	}
	defer raizSalida.Close()
	ctx := t.Context()
	pool, err := pgxpool.New(ctx, f.DSNPropietario)
	if err != nil {
		t.Fatal("ensayo_pg_no_disponible")
	}
	defer pool.Close()
	reloj := relojGobiernoUsuariosEnsayo{}
	switch f.Fase {
	case "verificar":
		informe, err := VerificarCadenaGobiernoUsuariosAdmin(ctx, pool, raizSalida, "verificacion-cadena.json")
		if err != nil {
			t.Fatal("ensayo_cadena_no_disponible")
		}
		if informe.Estado != "verificada" {
			t.Fatalf("ensayo_cadena_rechazada_%s", informe.Fallo.Codigo)
		}
	case "preparar":
		origen := MaterialOrigenGobiernoUsuariosAdmin{DirectorioMaterial: f.DirectorioMaterial, RutaConfiguracionHMAC: f.RutaConfiguracionHMAC, ArchivoSemillaRaiz: f.ArchivoSemillaRaiz, ValidezClaves: 2 * time.Hour}
		if _, err := PrepararGobiernoUsuariosAdmin(ctx, pool, origen, raizSalida, reloj); err != nil {
			t.Fatal("ensayo_proveedor_no_coincide_con_raiz")
		}
	default:
		operador, err := pgxpool.New(ctx, f.DSNOperador)
		if err != nil {
			t.Fatal("ensayo_login_ausente")
		}
		defer operador.Close()
		acuse, err := AplicarGobiernoUsuariosAdminPreparado(ctx, operador, raizSalida, "acuse-"+f.Fase+".json", reloj)
		if err != nil {
			t.Fatal("ensayo_aplicacion_sin_acuse_confirmado")
		}
		if acuse.Estado != "permitido" {
			t.Fatalf("ensayo_resultado_%s", acuse.Codigo)
		}
	}
}
