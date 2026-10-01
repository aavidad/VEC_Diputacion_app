package ensayofisicopg

import (
	"context"
	"errors"
	"os"
	"os/user"
	"strconv"
	"strings"
	"testing"

	puertos "vec-diputacion-granada/internal/modules/administracion/ports/ensayofisicopg"
)

type contextoFisicoPrueba struct {
	d   puertos.DatosContextoFisico
	err error
}

func (c *contextoFisicoPrueba) RevalidarContextoFisico(context.Context) (puertos.DatosContextoFisico, error) {
	return c.d, c.err
}
func TestPlantillasNoAceptaRuntimeDeclarado(t *testing.T) {
	ctx := context.Background()
	for _, e := range []puertos.Entorno{{}, {PostgreSQL: RuntimeObservacion{}}} {
		if _, err := NuevoRuntimePlantillas(ctx, e, &contextoFisicoPrueba{}); err == nil {
			t.Fatal("runtime no comprobado aceptado")
		}
	}
	if _, err := NuevoRuntimePlantillas(ctx, puertos.Entorno{}, nil); err == nil {
		t.Fatal("sin contexto")
	}
}
func TestNSSSinteticoPropio(t *testing.T) {
	raiz := t.TempDir()
	if err := prepararNSS(raiz); err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(raiz + "/passwd")
	if err != nil {
		t.Fatal(err)
	}
	esperado := "vecensayo:x:" + strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid()) + ":VEC:/tmp:/usr/sbin/nologin\n"
	if string(b) != esperado {
		t.Fatal("identidad NSS ajena")
	}
	if prepararNSS(raiz) == nil {
		t.Fatal("sobrescritura NSS aceptada")
	}
}
func comprobarNSSDentro() error {
	u, err := user.Current()
	if err != nil || u.Uid != strconv.Itoa(os.Getuid()) || u.Gid != strconv.Itoa(os.Getgid()) || u.HomeDir != "/tmp" || u.Username != "vecensayo" {
		return errors.New("nss_sintetico_no_comprobable")
	}
	return nil
}
func comprobarPlantillasReales(ctx context.Context, e puertos.Entorno, t *testing.T) error {
	t.Helper()
	fuente := &contextoFisicoPrueba{d: puertos.DatosContextoFisico{OperacionRef: "operacion:sintetica", ConjuntoRef: "conjunto:sintetico", VentanaRef: "ventana:sintetica", ManifiestoSHA256: strings.Repeat("a", 64), ArtefactoFisicoSHA256: e.Componentes[0].SHA256, ImagenSHA256: e.Aislamiento.ImagenSHA256}}
	p, err := NuevoRuntimePlantillas(ctx, e, fuente)
	if err != nil {
		return err
	}
	original, err := p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", "SELECT oid::text||'|'||datdba::text||'|'||datallowconn::text||'|'||coalesce(datacl::text,'') FROM pg_database WHERE datname='template0'"}, nil, 4096)
	if err != nil {
		return err
	}
	before, err := p.ObservarProcedenciaFisica(ctx)
	if err != nil {
		return err
	}
	if before.ArtefactoFisicoSHA256 != fuente.d.ArtefactoFisicoSHA256 {
		return errRuntime
	}
	if err = p.CrearClonPlantilla(ctx, "template0", p.NombreClon()); err != nil {
		return err
	}
	if _, err = p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", p.runtime.UsuarioBootstrap, "-d", p.NombreClon(), "-c", "SELECT count(*) FROM pg_namespace"}, nil, 4096); err != nil {
		return err
	}
	if err = p.RetirarClonPlantilla(ctx, p.NombreClon()); err != nil {
		return err
	}
	// CREATE realmente confirmado; la respuesta al controlador se pierde después.
	p.despuesTecnico = func(sql string, e error) error {
		if e == nil && strings.HasPrefix(sql, "CREATE DATABASE") {
			return context.DeadlineExceeded
		}
		return e
	}
	if p.CrearClonPlantilla(ctx, "template0", p.NombreClon()) == nil {
		return errors.New("respuesta_perdida_sin_error")
	}
	if m, err := p.leerClon(ctx); err != nil || m != nil {
		return errors.New("respuesta_perdida_sin_limpieza")
	}
	p.despuesTecnico = nil
	// Un nombre ajeno que aparece antes de CREATE se conserva, incluso si su
	// propietario coincide con la cuenta técnica de este mismo cluster sintético.
	if err = p.tecnico(ctx, `CREATE DATABASE "`+p.NombreClon()+`" WITH TEMPLATE template0`); err != nil {
		return err
	}
	if p.CrearClonPlantilla(ctx, "template0", p.NombreClon()) == nil || p.RetirarClonPlantilla(ctx, p.NombreClon()) == nil {
		return errors.New("clon_ajeno_aceptado")
	}
	if m, err := p.leerClon(ctx); err != nil || m == nil {
		return errors.New("clon_ajeno_eliminado")
	}
	// Esta retirada es teardown de la fixture, no la API de propiedad del ensayo.
	if err = p.tecnico(ctx, `DROP DATABASE "`+p.NombreClon()+`"`); err != nil {
		return err
	}
	after, err := p.runtime.EjecutarPostgreSQL(ctx, "psql", []string{"-X", "-At", "-U", p.runtime.UsuarioBootstrap, "-d", "postgres", "-c", "SELECT oid::text||'|'||datdba::text||'|'||datallowconn::text||'|'||coalesce(datacl::text,'') FROM pg_database WHERE datname='template0'"}, nil, 4096)
	if err != nil || string(after) != string(original) {
		return errors.New("template0_modificada")
	}
	// Renombrado real entre comprobación y DROP: el nombre desaparece, el OID
	// continúa vivo. Una respuesta ambigua no acredita retirada individual.
	if err = p.CrearClonPlantilla(ctx, "template0", p.NombreClon()); err != nil {
		return err
	}
	renombrado := p.NombreClon() + "_renombrado"
	p.antesTecnico = func(sql string) error {
		if strings.HasPrefix(sql, "DROP DATABASE") {
			p.antesTecnico = nil
			return p.tecnico(ctx, `ALTER DATABASE "`+p.NombreClon()+`" RENAME TO "`+renombrado+`"`)
		}
		return nil
	}
	if p.RetirarClonPlantilla(ctx, p.NombreClon()) == nil || p.propiedad == nil {
		return errors.New("renombrado_durante_drop_acreditado_como_retirada")
	}
	// Reconciliación/teardown de la fixture propia, sin ampliar la API del puente.
	if err = p.tecnico(ctx, `ALTER DATABASE "`+renombrado+`" RENAME TO "`+p.NombreClon()+`"`); err != nil {
		return err
	}
	if err = p.RetirarClonPlantilla(ctx, p.NombreClon()); err != nil {
		return err
	}
	// Revalidación de ventana/imagen/artefacto: no se sustituye por un booleano.
	fuente.d.VentanaRef = "ventana:otra"
	if _, err = p.ObservarProcedenciaFisica(ctx); err == nil {
		return errors.New("ventana_cambiada_aceptada")
	}
	fuente.d.VentanaRef = "ventana:sintetica"
	fuente.d.ArtefactoFisicoSHA256 = strings.Repeat("b", 64)
	if _, err = NuevoRuntimePlantillas(ctx, e, fuente); err == nil {
		return errors.New("huella_fabricada_aceptada")
	}
	return nil
}
