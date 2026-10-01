//go:build !cgo

// Package mlx provides an honest, compile-time fallback when the native MLX
// runtime is unavailable. It deliberately never pretends to execute tensors.
package mlx

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"
)

var ErrUnavailable = errors.New("MLX indisponível: este build não inclui CGO/native MLX")

const (
	End             = int(^uint32(0) >> 1)
	Nvfp4MaxProduct = 448 * 6
)

type DType int

const (
	DTypeBool DType = iota
	DTypeUint8
	DTypeUint16
	DTypeUint32
	DTypeUint64
	DTypeInt8
	DTypeInt16
	DTypeInt32
	DTypeInt64
	DTypeFloat16
	DTypeFloat32
	DTypeFloat64
	DTypeBFloat16
	DTypeComplex64
)

func (t DType) String() string              { return "unavailable" }
func (t *DType) UnmarshalJSON([]byte) error { return ErrUnavailable }

type (
	Array           struct{}
	Scope           struct{}
	Device          struct{}
	Stream          struct{}
	Memory          struct{}
	SafetensorsFile struct{}
	Byte            int
	KibiByte        int
	MebiByte        int
	GibiByte        int
	TebiByte        int
)

func (b Byte) String() string       { return fmt.Sprintf("%d B", b) }
func (b KibiByte) String() string   { return fmt.Sprintf("%d KiB", b) }
func (b MebiByte) String() string   { return fmt.Sprintf("%d MiB", b) }
func (b GibiByte) String() string   { return fmt.Sprintf("%d GiB", b) }
func (b TebiByte) String() string   { return fmt.Sprintf("%d TiB", b) }
func (Memory) LogValue() slog.Value { return slog.StringValue("MLX unavailable") }
func (Device) LogValue() slog.Value { return slog.StringValue("MLX unavailable") }
func (Stream) LogValue() slog.Value { return slog.StringValue("MLX unavailable") }

type slice struct{ args []int }

func Slice(args ...int) slice { return slice{args} }

type (
	CompileOption func(*compileConfig)
	compileConfig struct{}
	CompileFunc   func(...*Array) []*Array
)

func Shapeless() CompileOption          { return func(*compileConfig) {} }
func unavailable() *Array               { panic(ErrUnavailable) }
func unavailableN(_ ...*Array) []*Array { panic(ErrUnavailable) }
func Compile(_ string, _ CompileFunc, _ ...CompileOption) CompileFunc {
	return func(...*Array) []*Array { panic(ErrUnavailable) }
}

func Compile1(_ string, _ func(*Array) *Array, _ ...CompileOption) func(*Array) *Array {
	return func(*Array) *Array { panic(ErrUnavailable) }
}

func Compile2(_ string, _ func(*Array, *Array) *Array, _ ...CompileOption) func(*Array, *Array) *Array {
	return func(*Array, *Array) *Array { panic(ErrUnavailable) }
}

func Compile3(_ string, _ func(*Array, *Array, *Array) *Array, _ ...CompileOption) func(*Array, *Array, *Array) *Array {
	return func(*Array, *Array, *Array) *Array { panic(ErrUnavailable) }
}

var (
	GELU         = func(*Array) *Array { return unavailable() }
	GELUApprox   = GELU
	SiLU         = GELU
	ReLUSquared  = GELU
	SoftplusF32  = GELU
	SwiGLU       = func(*Array, *Array) *Array { return unavailable() }
	GeGLU        = SwiGLU
	LogitSoftcap = SwiGLU
)

func CheckInit() error                           { return ErrUnavailable }
func LoadedLibraryPath() (string, error)         { return "", ErrUnavailable }
func Version() string                            { return "unavailable" }
func GPUIsAvailable() bool                       { return false }
func MetalIsAvailable() bool                     { return false }
func CUDAIsAvailable() bool                      { return false }
func ClearCache()                                {}
func ResetPeakMemory()                           {}
func EnableCompile()                             {}
func DisableCompile()                            {}
func Eval(...*Array)                             {}
func AsyncEval(...*Array)                        {}
func ActiveMemory() int                          { return 0 }
func CacheMemory() int                           { return 0 }
func PeakMemory() int                            { return 0 }
func PrettyBytes(n int) fmt.Stringer             { return Byte(n) }
func DefaultDevice() Device                      { return Device{} }
func DefaultStream() Stream                      { return Stream{} }
func SetDefaultDeviceGPU()                       {}
func MaxRecommendedWorkingSetSize() (int, error) { return 0, ErrUnavailable }
func SetWiredLimit(int) (int, error)             { return 0, ErrUnavailable }
func New(string) *Array                          { return unavailable() }
func FromValue[T interface {
	~bool | ~int | ~float32 | ~float64 | ~complex64
}](T) *Array {
	return unavailable()
}

func FromValues[S ~[]E, E interface {
	~bool | ~uint8 | ~uint16 | ~uint32 | ~uint64 | ~int8 | ~int16 | ~int32 | ~int64 | ~float32 | ~float64 | ~complex64
}](S, ...int) *Array {
	return unavailable()
}
func NewArrayInt32([]int32, []int32) *Array                            { return unavailable() }
func NewScalarArray(float32) *Array                                    { return unavailable() }
func Zeros(DType, ...int) *Array                                       { return unavailable() }
func ZerosF32([]int32) *Array                                          { return unavailable() }
func (t *Array) String() string                                        { panic(ErrUnavailable) }
func (t *Array) LogValue() slog.Value                                  { return slog.StringValue("MLX unavailable") }
func (t *Array) DType() DType                                          { return DTypeFloat32 }
func (t *Array) Dim(int) int                                           { panic(ErrUnavailable) }
func (t *Array) Dims() []int                                           { panic(ErrUnavailable) }
func (t *Array) NumBytes() int                                         { panic(ErrUnavailable) }
func (t *Array) NumDims() int                                          { panic(ErrUnavailable) }
func (t *Array) Size() int                                             { panic(ErrUnavailable) }
func (t *Array) Int() int32                                            { panic(ErrUnavailable) }
func (t *Array) Float() float32                                        { panic(ErrUnavailable) }
func (t *Array) Ints() []int32                                         { panic(ErrUnavailable) }
func (t *Array) Floats() []float32                                     { panic(ErrUnavailable) }
func (t *Array) Save(string) error                                     { return ErrUnavailable }
func (t *Array) Set(*Array)                                            {}
func (t *Array) Clone() *Array                                         { return unavailable() }
func (t *Array) Slice(...slice) *Array                                 { return unavailable() }
func (t *Array) SliceUpdate(*Array, ...slice) *Array                   { return unavailable() }
func (t *Array) Abs() *Array                                           { return unavailable() }
func (t *Array) Add(*Array) *Array                                     { return unavailable() }
func (t *Array) Addmm(*Array, *Array, float32, float32) *Array         { return unavailable() }
func (t *Array) Argmax(int, bool) *Array                               { return unavailable() }
func (t *Array) ArgpartitionAxis(int, int) *Array                      { return unavailable() }
func (t *Array) ArgsortAxis(int) *Array                                { return unavailable() }
func (t *Array) AsStrided([]int, []int, int) *Array                    { return unavailable() }
func (t *Array) AsType(DType) *Array                                   { return unavailable() }
func (t *Array) BitwiseAnd(*Array) *Array                              { return unavailable() }
func (t *Array) BitwiseXor(*Array) *Array                              { return unavailable() }
func (t *Array) Categorical(int) *Array                                { return unavailable() }
func (t *Array) CategoricalWithKey(int, *Array) *Array                 { return unavailable() }
func (t *Array) Concatenate(int, ...*Array) *Array                     { return unavailable() }
func (t *Array) Cumsum(int, bool, bool) *Array                         { return unavailable() }
func (t *Array) Divide(*Array) *Array                                  { return unavailable() }
func (t *Array) Equal(*Array) *Array                                   { return unavailable() }
func (t *Array) ExpandDims(int) *Array                                 { return unavailable() }
func (t *Array) Flatten(int, int) *Array                               { return unavailable() }
func (t *Array) FloorDivide(*Array) *Array                             { return unavailable() }
func (t *Array) GatherMM(*Array, *Array, *Array, bool) *Array          { return unavailable() }
func (t *Array) Greater(*Array) *Array                                 { return unavailable() }
func (t *Array) Less(*Array) *Array                                    { return unavailable() }
func (t *Array) LessEqual(*Array) *Array                               { return unavailable() }
func (t *Array) LogsumexpAxis(int, bool) *Array                        { return unavailable() }
func (t *Array) Matmul(*Array) *Array                                  { return unavailable() }
func (t *Array) MaxAxis(int, bool) *Array                              { return unavailable() }
func (t *Array) Multiply(*Array) *Array                                { return unavailable() }
func (t *Array) Negative() *Array                                      { return unavailable() }
func (t *Array) Power(*Array) *Array                                   { return unavailable() }
func (t *Array) PutAlongAxis(*Array, *Array, int) *Array               { return unavailable() }
func (t *Array) Remainder(*Array) *Array                               { return unavailable() }
func (t *Array) Reshape(...int) *Array                                 { return unavailable() }
func (t *Array) RightShift(*Array) *Array                              { return unavailable() }
func (t *Array) ScatterAddAxis(*Array, *Array, int) *Array             { return unavailable() }
func (t *Array) Sigmoid() *Array                                       { return unavailable() }
func (t *Array) Sign() *Array                                          { return unavailable() }
func (t *Array) Sqrt() *Array                                          { return unavailable() }
func (t *Array) Squeeze(int) *Array                                    { return unavailable() }
func (t *Array) StackAxis(int, ...*Array) *Array                       { return unavailable() }
func (t *Array) Subtract(*Array) *Array                                { return unavailable() }
func (t *Array) SumAxis(int, bool) *Array                              { return unavailable() }
func (t *Array) TakeAlongAxis(*Array, int) *Array                      { return unavailable() }
func (t *Array) TakeAxis(*Array, int) *Array                           { return unavailable() }
func (t *Array) Tanh() *Array                                          { return unavailable() }
func (t *Array) Transpose(...int) *Array                               { return unavailable() }
func Add(a, b *Array) *Array                                           { return unavailable() }
func AddMM(c, a, b *Array, alpha, beta float32) *Array                 { return unavailable() }
func AddScalar(*Array, float32) *Array                                 { return unavailable() }
func Arange(float64, float64, float64, DType) *Array                   { return unavailable() }
func Argpartition(*Array, int, int) *Array                             { return unavailable() }
func Argsort(*Array, int) *Array                                       { return unavailable() }
func Bernoulli(*Array) *Array                                          { return unavailable() }
func BernoulliWithKey(*Array, *Array) *Array                           { return unavailable() }
func BroadcastTo(*Array, ...int32) *Array                              { return unavailable() }
func Clamp(*Array, float32, float32) *Array                            { return unavailable() }
func Clip(*Array, *Array, *Array) *Array                               { return unavailable() }
func Concatenate([]*Array, int) *Array                                 { return unavailable() }
func Contiguous(*Array, bool) *Array                                   { return unavailable() }
func Conv1d(*Array, *Array, *Array, int32, int32, int32, int32) *Array { return unavailable() }
func Conv2d(*Array, *Array, int32, int32, int32, int32, int32, int32, int32) *Array {
	return unavailable()
}
func Cos(*Array) *Array                                                  { return unavailable() }
func DepthwiseConv1d(*Array, *Array, *Array) *Array                      { return unavailable() }
func DepthwiseConvSiLU(*Array, *Array, *Array, int) *Array               { return unavailable() }
func Dequantize(*Array, *Array, *Array, int, int, string, *Array) *Array { return unavailable() }
func Div(*Array, *Array) *Array                                          { return unavailable() }
func DivScalar(*Array, float32) *Array                                   { return unavailable() }
func Erf(*Array) *Array                                                  { return unavailable() }
func Exp(*Array) *Array                                                  { return unavailable() }
func ExpandDims(*Array, int) *Array                                      { return unavailable() }
func FastScaledDotProductAttention(*Array, *Array, *Array, float32, string, *Array) *Array {
	return unavailable()
}
func Flatten(*Array) *Array                                { return unavailable() }
func FloorDivideScalar(*Array, int32) *Array               { return unavailable() }
func FromFP8(*Array, DType) *Array                         { return unavailable() }
func GLU(*Array) *Array                                    { return unavailable() }
func GatherMM(*Array, *Array, *Array, *Array, bool) *Array { return unavailable() }
func GatherQMM(*Array, *Array, *Array, *Array, *Array, *Array, bool, int, int, string, *Array, bool) *Array {
	return unavailable()
}
func LayerNormFn(*Array, *Array, *Array, float32) *Array         { return unavailable() }
func Log(*Array) *Array                                          { return unavailable() }
func Logaddexp(*Array, *Array) *Array                            { return unavailable() }
func Matmul(*Array, *Array) *Array                               { return unavailable() }
func Maximum(*Array, *Array) *Array                              { return unavailable() }
func Mean(*Array, int, bool) *Array                              { return unavailable() }
func Minimum(*Array, *Array) *Array                              { return unavailable() }
func Mul(*Array, *Array) *Array                                  { return unavailable() }
func MulScalar(*Array, float32) *Array                           { return unavailable() }
func Neg(*Array) *Array                                          { return unavailable() }
func Pad(*Array, []int, []int, []int, *Array, string) *Array     { return unavailable() }
func PadConstant(*Array, []int, []int, []int) *Array             { return unavailable() }
func Quantize(*Array, int, int, string) (*Array, *Array, *Array) { return nil, nil, nil }
func QuantizedMatmul(*Array, *Array, *Array, *Array, bool, int, int, string, *Array) *Array {
	return unavailable()
}

func RMSNormFn(*Array, *Array, float32) *Array { return unavailable() }

func RSqrt(*Array) *Array { return unavailable() }

func RandomKey(uint64) *Array { return unavailable() }

func ReLU(*Array) *Array { return unavailable() }

func Reshape(*Array, ...int32) *Array { return unavailable() }

func RoPEWithBase(*Array, int, bool, float32, float32, *Array) *Array { return unavailable() }

func RoPEWithFreqs(*Array, int, bool, float32, float32, *Array, *Array) *Array { return unavailable() }

func Sigmoid(*Array) *Array { return unavailable() }

func Sin(*Array) *Array { return unavailable() }

func SliceStartStop(*Array, []int32, []int32) *Array { return unavailable() }

func SoftmaxAxis(*Array, int, bool) *Array { return unavailable() }

func Softplus(*Array) *Array { return unavailable() }

func Squeeze(*Array, int) *Array { return unavailable() }

func Stack([]*Array, int) *Array { return unavailable() }

func Sub(*Array, *Array) *Array { return unavailable() }

func Sum(*Array, int, bool) *Array { return unavailable() }

func Take(*Array, *Array, int) *Array { return unavailable() }

func TakeAlongAxis(*Array, *Array, int) *Array { return unavailable() }

func Tile(*Array, []int32) *Array { return unavailable() }

func ToFP8(*Array) *Array { return unavailable() }

func Transpose(*Array, ...int) *Array { return unavailable() }

func Tri(int32, int32, int) *Array { return unavailable() }

func Where(*Array, *Array, *Array) *Array { return unavailable() }

func GatedDelta(*Array, *Array, *Array, *Array, *Array, *Array, bool) (*Array, *Array, []*Array) {
	return nil, nil, nil
}

func Mamba2Scan(*Array, *Array, *Array, *Array, *Array, *Array, *Array, *Array, *Array, bool) (*Array, *Array, []*Array) {
	return nil, nil, nil
}

func Load(string) iter.Seq2[string, *Array]                  { return func(yield func(string, *Array) bool) {} }
func LoadSafetensorsNative(string) (*SafetensorsFile, error) { return nil, ErrUnavailable }
func (s *SafetensorsFile) Free()                             {}
func (s *SafetensorsFile) Get(string) *Array                 { return unavailable() }
func (s *SafetensorsFile) GetMetadata(string) string         { return "" }
func SaveSafetensors(string, map[string]*Array) error        { return ErrUnavailable }
func SaveSafetensorsWithMetadata(string, map[string]*Array, map[string]string) error {
	return ErrUnavailable
}
func Collect(any) []*Array                          { return nil }
func NewScope() *Scope                              { return &Scope{} }
func (s *Scope) Attach(...*Array)                   {}
func (s *Scope) Close()                             {}
func (s *Scope) Detach(...*Array)                   {}
func (s *Scope) Discard(...*Array)                  {}
func Scoped(func())                                 {}
func ScopedArrays(func() []*Array) []*Array         { return nil }
func ScopedAsyncEval(func() []*Array) []*Array      { return nil }
func ScopedEval(func() []*Array) []*Array           { return nil }
func SigmoidRouter(*Array, *Array) (*Array, *Array) { return nil, nil }

type Linear struct{ Weight, Bias *Array }

func (*Linear) Forward(*Array) *Array                      { return unavailable() }
func (*Linear) Gather(*Array, *Array, *Array, bool) *Array { return unavailable() }

type Embedding struct{ Weight *Array }

func (*Embedding) Forward(*Array) *Array { return unavailable() }
func (e *Embedding) AsLinear() Linear    { return Linear{Weight: e.Weight} }

type LayerNorm struct{ Weight, Bias *Array }

func (*LayerNorm) Forward(*Array, float32) *Array { return unavailable() }

type RMSNorm struct{ Weight *Array }

func (*RMSNorm) Forward(*Array, float32) *Array { return unavailable() }
