package config

import (
	"testing"
	"time"
)

func TestValidezGobiernoAdminPorPerfil(t *testing.T) {
	desarrollo := Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}
	casos := []struct {
		nombre   string
		cfg      Config
		horas    int
		override string
		quiere   time.Duration
	}{
		{"defecto", Config{}, 0, "", 24 * time.Hour},
		{"minimo", Config{}, 1, "", time.Hour},
		{"produccion_24", Config{ExecutionProfile: ExecutionProfileProduction}, 24, "", 24 * time.Hour},
		{"produccion_25", Config{ExecutionProfile: ExecutionProfileProduction}, 25, "", 0},
		{"desarrollo_60_dias", desarrollo, 1440, "", 1440 * time.Hour},
		{"desarrollo_400_dias", desarrollo, 9600, "", 9600 * time.Hour},
		{"desarrollo_400_dias_mas_una_hora", desarrollo, 9601, "", 0},
		{"negativo", desarrollo, -1, "", 0},
		{"entorno_pisa_json", desarrollo, 2, "9600", 9600 * time.Hour},
		{"entorno_invalido", desarrollo, 2, "desconocido", 0},
		{"entorno_desbordado", desarrollo, 2, "999999999999999999999999999999", 0},
		{"entorno_cero_expreso", desarrollo, 2, "0", 0},
		{"perfil_ausente_25", Config{}, 25, "", 0},
		{"perfil_desconocido", Config{ExecutionProfile: "otro", AuthMode: AuthModeDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}, 1, "", 0},
		{"sin_modo", Config{ExecutionProfile: ExecutionProfileDevelopment, DevelopmentGuard: DevelopmentGuardAcknowledgement}, 25, "", 0},
		{"sin_guarda", Config{ExecutionProfile: ExecutionProfileDevelopment, AuthMode: AuthModeDevelopment}, 25, "", 0},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			got, err := tc.cfg.ValidezGobiernoAdmin(tc.horas, tc.override)
			if got != tc.quiere || (err != nil) != (tc.quiere == 0) {
				t.Fatalf("vigencia=%v error=%v; esperada=%v", got, err, tc.quiere)
			}
		})
	}
	if err := desarrollo.ValidarValidezGobiernoAdmin(time.Hour + time.Second); err == nil {
		t.Fatal("fraccion de hora aceptada")
	}
}
