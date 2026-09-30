package postgres

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"vec-diputacion-granada/internal/modules/usuarios/canonico"
	"vec-diputacion-granada/internal/modules/usuarios/domain"
	"vec-diputacion-granada/internal/modules/usuarios/ports"
	"vec-diputacion-granada/internal/shared/postgresql"
	vecdomain "vec-diputacion-granada/internal/vec/domain"
	vecports "vec-diputacion-granada/internal/vec/ports"
)

// El doble distingue la auditoría provisional de la confirmada: un aborto
// revierte la primera; sólo COMMIT confirmado conserva el acceso.
type txConsultaSerializablePrueba struct {
	acreditar                                                bool
	errAjuste, errACL, errConsulta, errCommit                error
	datos, foto                                              []byte
	ajustes, acls, consultas, commits, rollbacks, auditorias int
	sql                                                      string
	args                                                     []any
	cancelar                                                 context.CancelFunc
}

func (tx *txConsultaSerializablePrueba) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	tx.ajustes++
	return pgconn.CommandTag{}, tx.errAjuste
}
func (tx *txConsultaSerializablePrueba) QueryRow(_ context.Context, sql string, args ...any) filaCorreos {
	if strings.Contains(sql, "session_user=current_user") {
		tx.acls++
		return filaImagenPGPrueba{valores: []any{tx.acreditar}, err: tx.errACL}
	}
	tx.consultas++
	tx.sql, tx.args = sql, args
	if tx.cancelar != nil {
		tx.cancelar()
	}
	return filaImagenPGPrueba{valores: []any{tx.datos, tx.foto}, err: tx.errConsulta}
}
func (tx *txConsultaSerializablePrueba) Commit(context.Context) error {
	tx.commits++
	if tx.errCommit == nil {
		tx.auditorias++
	}
	return tx.errCommit
}
func (tx *txConsultaSerializablePrueba) Rollback(context.Context) error { tx.rollbacks++; return nil }

type txConsultaPreferenciasPrueba struct{ *txConsultaSerializablePrueba }

func (tx txConsultaPreferenciasPrueba) QueryRow(ctx context.Context, sql string, args ...any) filaPreferencias {
	return tx.txConsultaSerializablePrueba.QueryRow(ctx, sql, args...)
}

type casoConsultaSerializable struct {
	nombre                  string
	datos, foto             []byte
	indisponible, prohibido error
	preparar                func(func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error)
}

// Toda la matriz usa la superficie externa y sus audiencias exactas.
func casosConsultasSerializables(t *testing.T) []casoConsultaSerializable {
	t.Helper()
	superficie := vecdomain.SuperficieAutenticacionExternaPersonalV1
	ordenPref, pref, v3Pref := materialPrueba(t, ports.AccionConsultarPreferencias, superficie)
	actor, _ := ordenPref.ContextoActor()
	vinculo, _ := ordenPref.Vinculo()
	imagen := ports.MaterialImagen{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: ports.AccionConsultarImagen, FinalidadRef: ports.FinalidadImagenPropia, CatalogoVersionRef: domain.CatalogoBaseImagen().VersionRef}
	identidadImagen, err := domain.NuevaIdentidadImagen(actor, vinculo, superficie)
	if err != nil {
		t.Fatal(err)
	}
	ordenImagen := ports.OrdenImagen{Identidad: identidadImagen}
	recursoImagen, _ := canonico.RecursoImagen(imagen)
	huellaImagen, _ := recursoImagen.HuellaContextoAutorizacionSHA256()
	audienciaImagen, _ := canonico.AudienciaImagen(imagen.Accion, superficie)
	v3Imagen := v3ConsultaExternaPrueba(t, actor, imagen.Accion, huellaImagen, audienciaImagen)
	correo := ports.MaterialCorreos{Superficie: superficie, PersonaRef: actor.PersonaRef, PerfilRef: actor.PerfilActivoRef, Accion: ports.AccionConsultarCorreos, FinalidadRef: ports.FinalidadCorreosPropios}
	identidadCorreo, err := domain.NuevaIdentidadCorreos(actor, vinculo, superficie)
	if err != nil {
		t.Fatal(err)
	}
	ordenCorreo := ports.OrdenCorreos{Identidad: identidadCorreo}
	recursoCorreo, _ := canonico.RecursoCorreos(correo)
	huellaCorreo, _ := recursoCorreo.HuellaContextoAutorizacionSHA256()
	audienciaCorreo, _ := canonico.AudienciaCorreos(correo.Accion, superficie)
	v3Correo := v3ConsultaExternaPrueba(t, actor, correo.Accion, huellaCorreo, audienciaCorreo)
	serializar := func(v any) []byte {
		b, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	registroPref := func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) *RegistroPreferenciasPostgreSQL {
		return &RegistroPreferenciasPostgreSQL{superficie: superficie, rol: rolEjecutorPreferencias(superficie), iniciar: func(ctx context.Context) (transaccionPreferencias, error) {
			tx, err := iniciar(ctx)
			if err != nil {
				return nil, err
			}
			return txConsultaPreferenciasPrueba{tx}, nil
		}}
	}
	registroImagen := func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) *RegistroImagenPostgreSQL {
		return &RegistroImagenPostgreSQL{superficie: superficie, iniciar: func(ctx context.Context) (transaccionCorreos, error) { return iniciar(ctx) }}
	}
	return []casoConsultaSerializable{
		{nombre: "preferencias", datos: serializar(map[string]any{"existe": true, "persona_ref": actor.PersonaRef, "version": 2, "catalogo_version_ref": pref.CatalogoVersionRef, "valores": pref.Valores}), indisponible: ports.ErrNoDisponible, prohibido: ports.ErrProhibido, preparar: func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error) {
			r := registroPref(iniciar)
			return func(ctx context.Context) (bool, error) {
				estado, existe, err := r.ConsultarPropias(ctx, ordenPref, pref, v3Pref)
				return existe || !reflect.DeepEqual(estado, ports.EstadoPreferencias{}), err
			}
		}},
		{nombre: "catalogo_preferencias", datos: serializar(domain.CatalogoBasePreferencias()), indisponible: ports.ErrNoDisponible, prohibido: ports.ErrProhibido, preparar: func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error) {
			r := registroPref(iniciar)
			return func(ctx context.Context) (bool, error) {
				c, err := r.CatalogoVigente(ctx, ordenPref)
				return !reflect.DeepEqual(c, domain.CatalogoPreferencias{}), err
			}
		}},
		{nombre: "correos", datos: serializar(map[string]any{"persona_ref": actor.PersonaRef, "version": 1, "correos": []any{map[string]any{"correo_ref": refCorreoPG, "estado": "pendiente", "activo": false, "creado_utc": fechaPG(), "verificado_utc": nil, "sobre": sobrePG(), "codigo": nil}}}), indisponible: ports.ErrCorreosNoDisponible, prohibido: ports.ErrCorreosProhibido, preparar: func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error) {
			sentencias, _ := sentenciasCorreosSuperficie(superficie)
			r := &RegistroCorreosPostgreSQL{superficie: superficie, sql: sentencias, descifrador: &descifradorCorreoPGPrueba{}, iniciar: func(ctx context.Context) (transaccionCorreos, error) { return iniciar(ctx) }}
			return func(ctx context.Context) (bool, error) {
				vista, err := r.ConsultarPropios(ctx, ordenCorreo, correo, v3Correo)
				return !reflect.DeepEqual(vista, ports.VistaCorreos{}), err
			}
		}},
		{nombre: "imagen", datos: serializar(estadoImagenSQL{Existe: true, PersonaRef: actor.PersonaRef, Version: 2, CatalogoVersionRef: imagen.CatalogoVersionRef, Eleccion: domain.EleccionImagen{Modo: domain.ModoImagenFoto, Paleta: "verde"}}), foto: jpegPrueba, indisponible: ports.ErrImagenNoDisponible, prohibido: ports.ErrImagenProhibido, preparar: func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error) {
			r := registroImagen(iniciar)
			return func(ctx context.Context) (bool, error) {
				estado, existe, foto, err := r.Consultar(ctx, ordenImagen, imagen, v3Imagen)
				return existe || foto != nil || !reflect.DeepEqual(estado, ports.EstadoImagen{}), err
			}
		}},
		{nombre: "catalogo_imagen", datos: serializar(domain.CatalogoBaseImagen()), indisponible: ports.ErrImagenNoDisponible, prohibido: ports.ErrImagenProhibido, preparar: func(iniciar func(context.Context) (*txConsultaSerializablePrueba, error)) func(context.Context) (bool, error) {
			r := registroImagen(iniciar)
			return func(ctx context.Context) (bool, error) {
				c, err := r.CatalogoVigente(ctx, ordenImagen)
				return !reflect.DeepEqual(c, domain.CatalogoImagen{}), err
			}
		}},
	}
}

func v3ConsultaExternaPrueba(t *testing.T, actor vecdomain.ContextoActor, accion, huella, audiencia string) vecports.ExportacionMaterialConsumoAutorizacionAtestadaV3 {
	t.Helper()
	ahora := time.Now().UTC().Truncate(time.Microsecond)
	resumen, err := vecports.NuevoResumenCapacidadAtestacionAutorizacionV3("dec_prueba", strings.Repeat("a", 64), strings.Repeat("b", 64), "ctx_prueba", strings.Repeat("c", 64), accion, actor.PersonaRef, huella, audiencia, ahora, ahora.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	canon, _ := actor.RepresentacionCanonicaVinculadaV2()
	// SPKI Ed25519 sintética ya usada por las pruebas del adaptador.
	raiz := []byte{0x30, 0x2a, 0x30, 0x05, 0x06, 0x03, 0x2b, 0x65, 0x70, 0x03, 0x21, 0x00}
	raiz = append(raiz, bytes.Repeat([]byte{1}, 32)...)
	v3, err := vecports.NuevaExportacionMaterialConsumoAutorizacionAtestadaV3(bytes.Repeat([]byte("x"), 512), resumen, []byte("decision"), []byte("motivo"), canon, 1, 1, []byte("payload"), []byte("sobre"), []byte("evidencia"), raiz)
	if err != nil {
		t.Fatal(err)
	}
	return v3
}

func nuevaTxConsulta(c casoConsultaSerializable) *txConsultaSerializablePrueba {
	return &txConsultaSerializablePrueba{acreditar: true, datos: c.datos, foto: c.foto}
}

func TestConsultasSerializablesReabrenTransaccionCompleta(t *testing.T) {
	for _, caso := range casosConsultasSerializables(t) {
		for _, codigo := range []string{"40001", "40P01"} {
			for _, fase := range []string{"begin", "ajustes", "acl", "consulta", "commit"} {
				t.Run(caso.nombre+"/"+codigo+"/"+fase, func(t *testing.T) {
					primera, segunda := nuevaTxConsulta(caso), nuevaTxConsulta(caso)
					aborto := fmt.Errorf("abortada: %w", &pgconn.PgError{Code: codigo, Message: "detalle privado"})
					switch fase {
					case "ajustes":
						primera.errAjuste = aborto
					case "acl":
						primera.errACL = aborto
					case "consulta":
						primera.errConsulta = aborto
					case "commit":
						primera.errCommit = aborto
					}
					inicios := 0
					consultar := caso.preparar(func(context.Context) (*txConsultaSerializablePrueba, error) {
						inicios++
						if inicios == 1 {
							if fase == "begin" {
								return nil, aborto
							}
							return primera, nil
						}
						if primera.rollbacks != 1 && fase != "begin" {
							t.Fatal("la primera tx sigue abierta")
						}
						if inicios > 2 {
							t.Fatal("más de un reintento")
						}
						return segunda, nil
					})
					datos, err := consultar(context.Background())
					if err != nil || !datos || inicios != 2 || segunda.ajustes != 6 || segunda.acls != 1 || segunda.consultas != 1 || segunda.commits != 1 || segunda.rollbacks != 1 || segunda.auditorias != 1 || primera.auditorias != 0 {
						t.Fatalf("consulta: datos=%v err=%v inicios=%d tx=%+v", datos, err, inicios, segunda)
					}
					if primera.consultas > 0 && (!reflect.DeepEqual(primera.args, segunda.args) || primera.sql != segunda.sql) {
						t.Fatal("cambiaron actor/material/operación/finalidad entre intentos")
					}
				})
			}
		}
	}
}

func TestConsultasSerializablesDenegacionYCommitInciertoNoReintentan(t *testing.T) {
	for _, caso := range casosConsultasSerializables(t) {
		for _, fase := range []string{"consulta", "commit"} {
			for _, codigo := range []string{"42501", "55P03", "23505", "08006", "desconexion"} {
				t.Run(caso.nombre+"/"+fase+"/"+codigo, func(t *testing.T) {
					tx := nuevaTxConsulta(caso)
					var fallo error = &pgconn.PgError{Code: codigo, Message: "detalle privado"}
					if codigo == "desconexion" {
						fallo = errors.New("EOF: resultado COMMIT desconocido detalle privado")
					}
					if fase == "consulta" {
						tx.errConsulta = fallo
					} else {
						tx.errCommit = fallo
					}
					inicios := 0
					consultar := caso.preparar(func(context.Context) (*txConsultaSerializablePrueba, error) { inicios++; return tx, nil })
					datos, err := consultar(context.Background())
					esperado := caso.indisponible
					if codigo == "42501" {
						esperado = caso.prohibido
					}
					if datos || !errors.Is(err, esperado) || inicios != 1 || tx.rollbacks != 1 || postgresql.EsCarreraSerializable(err) || strings.Contains(err.Error(), "privado") {
						t.Fatalf("datos=%v err=%v inicios=%d", datos, err, inicios)
					}
				})
			}
		}
	}
}

func TestConsultasSerializablesRevalidanRevocacion(t *testing.T) {
	for _, caso := range casosConsultasSerializables(t) {
		for _, fase := range []string{"acl", "autorizacion"} {
			t.Run(caso.nombre+"/"+fase, func(t *testing.T) {
				primera, segunda := nuevaTxConsulta(caso), nuevaTxConsulta(caso)
				primera.errCommit = &pgconn.PgError{Code: "40001"}
				if fase == "acl" {
					segunda.acreditar = false
				} else {
					segunda.errConsulta = &pgconn.PgError{Code: "42501"}
				}
				inicios := 0
				consultar := caso.preparar(func(context.Context) (*txConsultaSerializablePrueba, error) {
					inicios++
					if inicios == 1 {
						return primera, nil
					}
					return segunda, nil
				})
				datos, err := consultar(context.Background())
				esperado := caso.indisponible
				if fase == "autorizacion" {
					esperado = caso.prohibido
				}
				if datos || !errors.Is(err, esperado) || inicios != 2 || segunda.acls != 1 || segunda.commits != 0 || primera.auditorias+segunda.auditorias != 0 || segunda.rollbacks != 1 {
					t.Fatalf("revocación: datos=%v err=%v inicios=%d", datos, err, inicios)
				}
				if fase == "acl" && segunda.consultas != 0 {
					t.Fatal("consulta tras revocar ejecutor")
				}
			})
		}
	}
}

func TestConsultasSerializablesCancelacionYAgotamiento(t *testing.T) {
	for _, caso := range casosConsultasSerializables(t) {
		t.Run(caso.nombre+"/cancelacion", func(t *testing.T) {
			ctx, cancelar := context.WithCancel(context.Background())
			defer cancelar()
			tx := nuevaTxConsulta(caso)
			tx.errConsulta, tx.cancelar = &pgconn.PgError{Code: "40001"}, cancelar
			inicios := 0
			consultar := caso.preparar(func(context.Context) (*txConsultaSerializablePrueba, error) { inicios++; return tx, nil })
			datos, err := consultar(ctx)
			if datos || !errors.Is(err, context.Canceled) || inicios != 1 || tx.rollbacks != 1 {
				t.Fatalf("cancelación: datos=%v err=%v inicios=%d", datos, err, inicios)
			}
		})
		t.Run(caso.nombre+"/agotamiento", func(t *testing.T) {
			inicios := 0
			var anterior *txConsultaSerializablePrueba
			consultar := caso.preparar(func(context.Context) (*txConsultaSerializablePrueba, error) {
				if anterior != nil && anterior.rollbacks != 1 {
					t.Fatal("tx anterior abierta")
				}
				inicios++
				anterior = nuevaTxConsulta(caso)
				anterior.errConsulta = &pgconn.PgError{Code: "40P01", Message: "detalle privado"}
				return anterior, nil
			})
			datos, err := consultar(context.Background())
			if datos || !errors.Is(err, caso.indisponible) || inicios != postgresql.IntentosMaximosCarreraSerializable || anterior.rollbacks != 1 || postgresql.EsCarreraSerializable(err) {
				t.Fatalf("agotamiento: datos=%v err=%v inicios=%d", datos, err, inicios)
			}
		})
	}
}

func TestConsultaCanceladaDuranteEsperaNoExponeMarcador(t *testing.T) {
	ctx, cancelar := context.WithCancel(context.Background())
	err := errorConsultaSerializable(ctx, &pgconn.PgError{Code: "40001"}, errorSeguro)
	cancelar()
	if obtenido := errorFinalConsultaSerializable(ctx, err); !errors.Is(obtenido, context.Canceled) || postgresql.EsCarreraSerializable(obtenido) {
		t.Fatalf("%v", obtenido)
	}
}
