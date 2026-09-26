package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"vec-diputacion-granada/internal/modules/seleccion/adapters/catalogo"
	"vec-diputacion-granada/internal/modules/seleccion/domain"
	"vec-diputacion-granada/internal/modules/seleccion/ports"
	"vec-diputacion-granada/internal/shared/baremacion"
)

// materialDoble es el material que aceptan los dobles de AD3-89/AD3-90 de
// deploy/postgresql/seleccion/pruebas_sql/dobles_ad3_89_90.sql: operación,
// recurso y persona del contexto. La criptografía real se prueba aparte.
type materialDoble struct {
	operacion, recurso, persona string
}

func (m materialDoble) CapacidadCanonica() []byte {
	b, _ := json.Marshal(map[string]string{"operacion": m.operacion, "efecto_ref": m.recurso})
	return b
}
func (m materialDoble) DecisionCanonica() []byte {
	b, _ := json.Marshal(map[string]string{"accion": m.operacion, "recurso_ref": m.recurso})
	return b
}
func (materialDoble) MotivoCanonico() []byte { return []byte{0} }
func (m materialDoble) ContextoActorCanonico() []byte {
	b, _ := json.Marshal(map[string]string{"persona_ref": m.persona})
	return b
}
func (materialDoble) PersonaVersion() uint64        { return 1 }
func (materialDoble) PerfilVersion() uint64         { return 1 }
func (materialDoble) PayloadVECAD3() []byte         { return []byte{0} }
func (materialDoble) SobreCOSESign1() []byte        { return []byte{0} }
func (materialDoble) EvidenciaVerificacion() []byte { return []byte{0} }
func (materialDoble) RaizPublicaSPKI() []byte       { return []byte{0} }

// Requiere un PostgreSQL 18 con roles de Selección, AD3-89/90, Selección
// 000001 y los dobles de las fachadas, y un LOGIN miembro solo de
// vec_seleccion_ejecutor (VEC_SELECCION_PG_DSN). Lo prepara
// deploy/postgresql/seleccion/probar_pg18.sh.
func TestRepositorioPostgreSQLRecorridoCompleto(t *testing.T) {
	dsn := os.Getenv("VEC_SELECCION_PG_DSN")
	if dsn == "" {
		t.Skip("requiere VEC_SELECCION_PG_DSN (PostgreSQL 18 desechable con Selección 000001 y dobles AD3)")
	}
	ctx, cancelar := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancelar()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	estado, err := ComprobarMigraciones(ctx, pool)
	if err != nil || !estado.AD389 || !estado.AD390 || !estado.Seleccion1 {
		t.Fatalf("migraciones no detectadas: %+v %v", estado, err)
	}
	repo, err := NuevoRepositorio(pool)
	if err != nil {
		t.Fatal(err)
	}
	// Publicación del catálogo del repositorio con un plazo abierto ahora:
	// doble arranque sin versiones nuevas.
	fuente, err := catalogo.NuevoFichero("../../../../../data/demo/reglas/seleccion_convocatorias.ejemplo.demo.json")
	if err != nil {
		t.Fatal(err)
	}
	convocatorias, _ := fuente.Convocatorias(ctx)
	c := convocatorias[0]
	c.Ref = "ensayo-go-" + time.Now().UTC().Format("20060102150405.000000")
	c.Ref = strings.ReplaceAll(c.Ref, ".", "-")
	c.AbreEn, c.CierraEn = time.Now().UTC().Add(-time.Hour).Truncate(time.Second), time.Now().UTC().Add(time.Hour).Truncate(time.Second)
	for i, esperadaNueva := range []bool{true, false} {
		version, nueva, err := repo.PublicarConvocatoria(ctx, c)
		if err != nil || version != 1 || nueva != esperadaNueva {
			t.Fatalf("arranque %d: versión %d nueva %v err %v", i+1, version, nueva, err)
		}
	}
	vigentes, err := repo.ConvocatoriasVigentes(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var publicada domain.ConvocatoriaPublicada
	for _, v := range vigentes {
		if v.Ref == c.Ref {
			publicada = v
		}
	}
	if !publicada.Abierta || publicada.Version != 1 || len(publicada.Requisitos) != len(c.Requisitos) {
		t.Fatalf("convocatoria publicada inesperada: %+v", publicada)
	}
	h1, _ := c.HuellaSHA256()
	if h2, _ := publicada.HuellaSHA256(); h1 != h2 {
		t.Fatal("la lectura no reconstruye la publicación")
	}

	persona := "per_" + strings.Repeat("G", 22) + strings.ReplaceAll(time.Now().UTC().Format("150405.000000"), ".", "")
	propio := func(op string) ports.MaterialConsumoV3 {
		return materialDoble{operacion: op, recurso: ports.PrefijoRecursoPropio + persona, persona: persona}
	}
	punt, _ := baremacion.PuntosDesdeMicropuntos(1_400_000)
	comando := ports.ComandoGuardarBorrador{PersonaRef: persona, ConvocatoriaRef: c.Ref, ConvocatoriaVersion: 1, Clave: "clave-go-0001",
		HuellaMaterial: strings.Repeat("1", 64), SolicitudRefNueva: "sol_" + strings.Repeat("g", 30),
		Sobre:      ports.SobreDatos{ClaveRef: "clave:prueba", Nonce: make([]byte, 12), Cifrado: []byte("sobre-cifrado-sintetico")},
		Requisitos: []domain.RequisitoDeclarado{{Clave: "nacionalidad", Estado: domain.RequisitoCumple}},
		Puntuacion: punt, Material: propio(ports.AccionGuardarBorrador)}
	parcial, err := repo.GuardarBorrador(ctx, comando)
	if err != nil || parcial.Version != 1 || parcial.DatosCompletos {
		t.Fatalf("borrador parcial: %+v %v", parcial, err)
	}
	comando.Clave, comando.HuellaMaterial, comando.VersionEsperada = "clave-go-0002", strings.Repeat("2", 64), 1
	comando.Turno, comando.DocumentoHuella, comando.DocumentoParcial, comando.DatosCompletos = "libre", strings.Repeat("d", 64), "***5678*", true
	completo, err := repo.GuardarBorrador(ctx, comando)
	if err != nil || completo.Version != 2 || !completo.DatosCompletos || completo.SolicitudRef != parcial.SolicitudRef {
		t.Fatalf("borrador completo: %+v %v", completo, err)
	}
	if repetido, err := repo.GuardarBorrador(ctx, comando); err != nil || !repetido.Reutilizada || repetido.Version != 2 {
		t.Fatalf("repetición: %+v %v", repetido, err)
	}
	comando.HuellaMaterial = strings.Repeat("3", 64)
	if _, err := repo.GuardarBorrador(ctx, comando); !errors.Is(err, ports.ErrClaveReutilizada) {
		t.Fatalf("material cambiado: %v", err)
	}
	comando.Clave = "clave-go-0003"
	if _, err := repo.GuardarBorrador(ctx, comando); !errors.Is(err, ports.ErrVersionObsoleta) {
		t.Fatalf("versión obsoleta: %v", err)
	}
	leido, err := repo.LeerBorrador(ctx, persona, c.Ref, propio(ports.AccionConsultarPropias))
	if err != nil || leido.Version != 2 || leido.Turno != "libre" || !leido.DatosCompletos || string(leido.Sobre.Cifrado) != "sobre-cifrado-sintetico" {
		t.Fatalf("lectura: %+v %v", leido, err)
	}
	if _, err := repo.LeerBorrador(ctx, persona, "otra-convocatoria", propio(ports.AccionConsultarPropias)); !errors.Is(err, ports.ErrSinBorrador) {
		t.Fatalf("sin borrador: %v", err)
	}
	presentar := ports.ComandoPresentar{PersonaRef: persona, SolicitudRef: completo.SolicitudRef, VersionEsperada: 2, Clave: "clave-go-pres-1",
		HuellaMaterial: strings.Repeat("4", 64), ReciboRef: "recibo:seleccion-presentacion:" + strings.Repeat("5", 64), Material: propio(ports.AccionPresentar)}
	recibo, err := repo.Presentar(ctx, presentar)
	if err != nil || !strings.Contains(recibo.NumeroJustificante, "/SOL-") || recibo.Reutilizada {
		t.Fatalf("presentación: %+v %v", recibo, err)
	}
	if otra, err := repo.Presentar(ctx, presentar); err != nil || !otra.Reutilizada || otra.NumeroJustificante != recibo.NumeroJustificante || !otra.PresentadaEn.Equal(recibo.PresentadaEn) {
		t.Fatalf("repetición de la presentación: %+v %v", otra, err)
	}
	ajena := presentar
	ajena.PersonaRef = "per_" + strings.Repeat("H", 30)
	ajena.Material = materialDoble{operacion: ports.AccionPresentar, recurso: ports.PrefijoRecursoPropio + ajena.PersonaRef, persona: ajena.PersonaRef}
	if _, err := repo.Presentar(ctx, ajena); !errors.Is(err, ports.ErrSolicitudNoEncontrada) {
		t.Fatalf("otra persona presentó una solicitud ajena: %v", err)
	}
	propias, err := repo.ListarPropias(ctx, persona, propio(ports.AccionConsultarPropias))
	if err != nil || len(propias) != 1 || propias[0].Estado != domain.EstadoPresentada || propias[0].NumeroJustificante != recibo.NumeroJustificante {
		t.Fatalf("mis solicitudes: %+v %v", propias, err)
	}
	rrhh := "per_" + strings.Repeat("R", 24)
	listado := materialDoble{operacion: ports.AccionConsultarSolicitudes, persona: rrhh,
		recurso: ports.PrefijoRecursoConvocatoria + huellaConvocatoria(c.Ref)}
	filas, err := repo.ListarPresentadas(ctx, c.Ref, 0, 10, listado)
	if err != nil || len(filas) != 1 || filas[0].DocumentoParcial != "***5678*" || filas[0].PersonaRef != persona {
		t.Fatalf("listado RRHH: %+v %v", filas, err)
	}
	ficha, err := repo.LeerFicha(ctx, completo.SolicitudRef, materialDoble{operacion: ports.AccionConsultarDetalle, persona: rrhh,
		recurso: ports.PrefijoRecursoSolicitud + completo.SolicitudRef})
	if err != nil || ficha.Fila.NumeroJustificante != recibo.NumeroJustificante || len(ficha.Historia) != 3 || ficha.Convocatoria.Version != 1 {
		t.Fatalf("ficha RRHH: %+v %v", ficha, err)
	}
}

func huellaConvocatoria(ref string) string {
	suma := sha256.Sum256([]byte(ref))
	return hex.EncodeToString(suma[:])
}
