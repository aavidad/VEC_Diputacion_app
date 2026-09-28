package confianzaatestacion

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"
	"time"

	usuariosdomain "vec-diputacion-granada/internal/modules/usuarios/domain"
	usuariosports "vec-diputacion-granada/internal/modules/usuarios/ports"
	core "vec-diputacion-granada/internal/vec/domain"
)

// Atraviesa el emisor HMAC real con una decisión y una atestación reales de
// prueba, usando exactamente la preimagen de Usuarios que reconstruye SQL.
func TestEmisorRealCapacidadV3UsuariosPreferencias(t *testing.T) {
	base := nuevoEscenarioConfianzaAtestacionV3Prueba(t)
	for _, caso := range []struct {
		accion, audiencia string
		campos            []string
	}{
		{usuariosports.AccionConsultarPreferencias, "vec_usuarios.preferencias.consultar.v1", []string{"catalogo", "valores", "version"}},
		{usuariosports.AccionActualizarPreferencias, "vec_usuarios.preferencias.actualizar.v1", []string{"valores", "version"}},
	} {
		t.Run(caso.accion, func(t *testing.T) {
			m := usuariosports.MaterialPreferencias{
				PersonaRef: base.resultado.Contexto.PersonaRef, PerfilRef: base.resultado.Contexto.PerfilActivoRef,
				Accion: caso.accion, FinalidadRef: usuariosports.FinalidadPreferenciasPropias,
				CatalogoVersionRef: "usuarios-preferencias-v1", VersionEsperada: 0,
				ClaveOperacion: "operacion-1234567890", HuellaPeticion: strings.Repeat("a", 64),
				Valores: usuariosdomain.CatalogoBasePreferencias().Predeterminados,
			}
			b, err := json.Marshal(m)
			if err != nil {
				t.Fatal(err)
			}
			h := sha256.Sum256(b)
			recurso := core.RecursoAutorizable{Referencia: m.PersonaRef, ModuloID: "usuarios", Tipo: "preferencias_persona", Ambitos: map[string]string{"persona_ref": m.PersonaRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(h[:])}}
			datosBase, err := base.solicitud.Datos()
			if err != nil {
				t.Fatal(err)
			}
			solicitud, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
				VinculoAutenticacionActor: datosBase.VinculoAutenticacionActor, ReferenciaMotivo: base.motivo, Accion: caso.accion, Recurso: recurso,
				Finalidad: usuariosports.FinalidadPreferenciasPropias, Correlacion: datosBase.Correlacion,
			})
			if err != nil {
				t.Fatal(err)
			}
			vinculo, err := datosBase.VinculoAutenticacionActor.Datos()
			if err != nil {
				t.Fatal(err)
			}
			version := core.VersionRol{RolID: "titular_preferencias", Version: 1, Nombre: "Titular preferencias", Estado: core.EstadoVersionRolPublicada,
				Concesiones:  []core.ConcesionRol{{Accion: caso.accion, ModuloID: "usuarios", TipoRecurso: "preferencias_persona", Finalidades: []string{usuariosports.FinalidadPreferenciasPropias}, GarantiaMinima: core.AuthAssuranceSubstantial, CamposPermitidos: caso.campos}},
				PublicadaPor: "responsable-seguridad", PublicadaEn: base.ahora.Add(-24 * time.Hour)}
			huellaCatalogo, err := core.HuellaCatalogoPoliticasAutorizacion(nil)
			if err != nil {
				t.Fatal(err)
			}
			instantanea := core.InstantaneaAutorizacion{AsignacionPerfil: core.AsignacionPerfil{AsignacionID: "asig-usuarios-preferencias", Version: 1, PerfilActivoRef: vinculo.PerfilActivoRef, PrincipalID: vinculo.PrincipalID,
				Ambitos:       []core.AmbitoPerfil{{Clave: "persona_ref", Valores: []string{m.PersonaRef}}},
				VersionRolRef: version.Referencia(), Estado: core.EstadoAsignacionPerfilActiva, VigenteDesde: base.ahora.Add(-time.Hour), VigenteHasta: base.ahora.Add(time.Hour), EmitidaPor: "administrador-identidades", EmitidaEn: base.ahora.Add(-2 * time.Hour)},
				VersionRol: version, ControlVigenciaVersionRol: core.ControlVigenciaVersionRol{VersionRolRef: version.Referencia(), Revision: 1, Estado: core.EstadoControlVigenciaVersionRolHabilitada, ActualizadoPor: version.PublicadaPor, ActualizadoEn: version.PublicadaEn},
				RevisionCatalogoPoliticas: 1, CatalogoPoliticasHuellaSHA256: huellaCatalogo}
			evidencia, err := core.NuevaEvidenciaEvaluacionAutorizacionV3(solicitud, instantanea, "dec_0123456789abcdef0123456789abcdef", base.ahora, base.ahora.Add(90*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			decision, err := core.NuevaDecisionAutorizacionLigadaV3(solicitud, evidencia)
			if err != nil {
				t.Fatal(err)
			}
			cabecera := core.CabeceraAtestacionAutorizacionV3{FormatoVersion: core.VersionFormatoAtestacionAutorizacionV3, Suite: SuiteAtestacionAutorizacionV3COSEEdDSA, ClaveID: base.raiz.claveID, Audiencia: audienciaConfianzaAtestacionV3Prueba}
			atestacion := atestacionConfianzaAtestacionV3Prueba(t, cabecera, decision, base.motivo, base.resultado, base.privada, base.ahora)
			prueba, err := base.servicio.Verificar(context.Background(), solicitud, decision, base.motivo, base.resultado, atestacion)
			if err != nil {
				t.Fatal(err)
			}
			clave, err := NuevaClaveHMACCapacidadAtestacionAutorizacionV3("clave:capacidad:usuarios-preferencias:v1", 1, bytes.Repeat([]byte{0x51}, 32), "emisor:usuarios-preferencias:v1", caso.audiencia, EstadoClaveHMACCapacidadAtestacionV3Emision, base.ahora.Add(-time.Hour), base.ahora.Add(time.Hour), time.Time{}, 7, strings.Repeat("7", 64))
			if err != nil {
				t.Fatal(err)
			}
			emisor, err := nuevoEmisorCapacidadesAtestacionAutorizacionV3(clave, &relojConfianzaAtestacionV3Prueba{ahora: base.ahora.Add(time.Microsecond)}, bytes.NewReader(append(bytes.Repeat([]byte{0x91}, 32), bytes.Repeat([]byte{0x92}, 32)...)))
			if err != nil {
				t.Fatal(err)
			}
			capacidad, err := emisor.Emitir(context.Background(), solicitud, decision, base.motivo, base.resultado, atestacion, prueba)
			if err != nil {
				t.Fatal(err)
			}
			segunda, err := emisor.Emitir(context.Background(), solicitud, decision, base.motivo, base.resultado, atestacion, prueba)
			if err != nil {
				t.Fatal(err)
			}
			primeraBytes, err := capacidad.ExportacionCanonicaParaConsumidor()
			if err != nil {
				t.Fatal(err)
			}
			segundaBytes, err := segunda.ExportacionCanonicaParaConsumidor()
			if err != nil || bytes.Equal(primeraBytes, segundaBytes) {
				t.Fatal("dos emisiones reutilizan la misma capacidad V3")
			}
			resumen, err := capacidad.ResumenParaConsumidor()
			if err != nil {
				t.Fatal(err)
			}
			huellaRecurso, err := recurso.HuellaContextoAutorizacionSHA256()
			if err != nil {
				t.Fatal(err)
			}
			if resumen.AudienciaConsumo() != caso.audiencia || resumen.Operacion() != caso.accion || resumen.EfectoRef() != m.PersonaRef || resumen.EfectoHuellaSHA256() != huellaRecurso {
				t.Fatal("capacidad real no liga acción, persona, audiencia y preimagen")
			}
			for indice, mutar := range []func(*usuariosports.MaterialPreferencias){
				func(x *usuariosports.MaterialPreferencias) { x.PersonaRef = "per_otra_persona_0123456789" },
				func(x *usuariosports.MaterialPreferencias) { x.PerfilRef = "prf_otro_perfil_0123456789" },
				func(x *usuariosports.MaterialPreferencias) { x.CatalogoVersionRef = "usuarios-preferencias-v2" },
				func(x *usuariosports.MaterialPreferencias) { x.Valores.Tema = "oscuro" },
			} {
				copia := m
				mutar(&copia)
				bytesMutados, err := json.Marshal(copia)
				if err != nil {
					t.Fatal(err)
				}
				hm := sha256.Sum256(bytesMutados)
				recursoMutado := core.RecursoAutorizable{Referencia: copia.PersonaRef, ModuloID: "usuarios", Tipo: "preferencias_persona", Ambitos: map[string]string{"persona_ref": copia.PersonaRef}, Atributos: map[string]string{"material_sha256": hex.EncodeToString(hm[:])}}
				huellaMutada, err := recursoMutado.HuellaContextoAutorizacionSHA256()
				if err != nil || huellaMutada == resumen.EfectoHuellaSHA256() {
					t.Fatal("capacidad previa serviría para otro material")
				}
				if indice == 0 && instantanea.AsignacionPerfil.Cubre(recursoMutado) {
					t.Fatal("la asignación propia cubriría otra persona")
				}
				solicitudMutada, err := core.NuevaSolicitudAutorizacionLigadaV3(core.DatosSolicitudAutorizacionLigadaV3{
					VinculoAutenticacionActor: datosBase.VinculoAutenticacionActor, ReferenciaMotivo: base.motivo,
					Accion: caso.accion, Recurso: recursoMutado, Finalidad: usuariosports.FinalidadPreferenciasPropias,
					Correlacion: datosBase.Correlacion,
				})
				if err != nil || decision.ValidarPara(solicitudMutada) == nil {
					t.Fatal("la decisión firmada admite un ámbito o material alterado")
				}
			}
		})
	}
}
