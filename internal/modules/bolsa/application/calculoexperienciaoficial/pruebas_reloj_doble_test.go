package calculoexperienciaoficial

import "time"

type relojSecuencialPrueba struct {
	actual time.Time
	paso   time.Duration
}

func (r *relojSecuencialPrueba) Ahora() time.Time {
	actual := r.actual
	r.actual = r.actual.Add(r.paso)
	return actual
}
