## Template Test

Hello! This template has been compiled on {{ .OS.OS }}-{{ .OS.Arch }} using {{ .OS.RuntimeVersion }}

I have some Env variables:

* {{ .ENV.FOO_BAR }} exported as `ENV_FOO_BAR`

Also context args such as {{ .App.Foo }} and {{ .App.Bar }}. And collections, see other files.
