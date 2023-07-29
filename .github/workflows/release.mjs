let bin = 'walk'
let goos = [
  'linux',
  'darwin',
  'windows',
]
let goarch = [
  'amd64',
  'arm64',
]

let name = (GOOS, GOARCH) => `${bin}_${GOOS}_${GOARCH}` + (GOOS === 'windows' ? '.exe' : '')

await $`go mod download`

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`GOOS=${GOOS} GOARCH=${GOARCH} go build -o ${name(GOOS, GOARCH)}`)))

await Promise.all(
  goos.flatMap(GOOS =>
    goarch.map(GOARCH =>
      $`gh release upload ${process.env.RELEASE_VERSION} ${name(GOOS, GOARCH)}`)))
