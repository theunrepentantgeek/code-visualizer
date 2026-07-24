package config

// Spiral holds persistent configuration for spiral timeline visualizations.
// All fields are pointers: nil means the field was not configured, non-nil
// means it was explicitly set (by a config file or by a CLI flag override).
type Spiral struct {
	Resolution    *string     `yaml:"resolution,omitempty"        json:"resolution,omitempty"`
	Size          *string     `yaml:"size,omitempty"              json:"size,omitempty"`
	Fill          *MetricSpec `yaml:"fill,omitempty"              json:"fill,omitempty"`
	Border        *MetricSpec `yaml:"border,omitempty"            json:"border,omitempty"`
	Surface       *bool       `yaml:"surface,omitempty"           json:"surface,omitempty"`
	SurfaceMetric *MetricSpec `yaml:"surfaceMetric,omitempty"  json:"surfaceMetric,omitempty"`
	Labels        *string     `yaml:"labels,omitempty"            json:"labels,omitempty"`
}

// OverrideResolution sets Resolution to v if v is non-empty.
func (s *Spiral) OverrideResolution(v string) { overrideString(&s.Resolution, v) }

// OverrideSize sets Size to v if v is non-empty.
func (s *Spiral) OverrideSize(v string) { overrideString(&s.Size, v) }

// OverrideFill sets Fill to v if v is non-zero.
func (s *Spiral) OverrideFill(v MetricSpec) { overrideMetricSpec(&s.Fill, v) }

// OverrideBorder sets Border to v if v is non-zero.
func (s *Spiral) OverrideBorder(v MetricSpec) { overrideMetricSpec(&s.Border, v) }

// OverrideSurface enables Surface when v is true.
func (s *Spiral) OverrideSurface(v bool) {
	if v {
		s.Surface = &v
	}
}

// OverrideSurfaceMetric sets SurfaceMetric to v if v is non-zero.
func (s *Spiral) OverrideSurfaceMetric(v MetricSpec) { overrideMetricSpec(&s.SurfaceMetric, v) }

// OverrideLabels sets Labels to v if v is non-empty.
func (s *Spiral) OverrideLabels(v string) { overrideString(&s.Labels, v) }

// SurfaceEnabled reports whether the spiral surface has been configured.
func (s *Spiral) SurfaceEnabled() bool {
	return s != nil && ((s.Surface != nil && *s.Surface) ||
		(s.SurfaceMetric != nil && !s.SurfaceMetric.IsZero()))
}
