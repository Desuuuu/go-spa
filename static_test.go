package spa_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Desuuuu/go-spa"
	"github.com/stretchr/testify/require"
)

func Example() {
	http.Handle("/", spa.StaticHandler(http.Dir("/static")))
}

func TestStaticHandler(t *testing.T) {
	testFS := http.Dir("./testdata")

	t.Run("serve existing files", func(t *testing.T) {
		h := spa.StaticHandler(testFS)

		res := makeRequest(h, http.MethodGet, "/test.css")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/test.css"), readAll(res.Body))

		res = makeRequest(h, http.MethodGet, "/dir/test.js")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/dir/test.js"), readAll(res.Body))
	})

	t.Run("serve fallback for non-existing files", func(t *testing.T) {
		h := spa.StaticHandler(testFS)

		res := makeRequest(h, http.MethodGet, "/test.js")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/index.html"), readAll(res.Body))
	})

	t.Run("serve fallback for directories", func(t *testing.T) {
		h := spa.StaticHandler(testFS)

		res := makeRequest(h, http.MethodGet, "/dir")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/index.html"), readAll(res.Body))

		res = makeRequest(h, http.MethodGet, "/dir/")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/index.html"), readAll(res.Body))

		res = makeRequest(h, http.MethodGet, "/test.css/")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/index.html"), readAll(res.Body))
	})

	t.Run("index redirect", func(t *testing.T) {
		h := spa.StaticHandler(testFS)

		res := makeRequest(h, http.MethodGet, "/index.html?query#fragment")
		require.Equal(t, http.StatusMovedPermanently, res.StatusCode)
		require.Equal(t, "./?query#fragment", res.Header.Get("Location"))
	})

	t.Run("no index redirect", func(t *testing.T) {
		h := spa.StaticHandler(testFS, spa.NoIndexRedirect())

		res := makeRequest(h, http.MethodGet, "/index.html?query#fragment")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/index.html"), readAll(res.Body))
	})

	t.Run("custom fallback", func(t *testing.T) {
		h := spa.StaticHandler(testFS, spa.Fallback("/dir/test.js"))

		res := makeRequest(h, http.MethodGet, "/")
		require.Equal(t, http.StatusOK, res.StatusCode)
		require.Equal(t, readFile(testFS, "/dir/test.js"), readAll(res.Body))
	})
}

func readFile(fs http.FileSystem, name string) string {
	file, err := fs.Open(name)
	if err != nil {
		panic(err)
	}

	return readAll(file)
}

func readAll(r io.Reader) string {
	data, err := io.ReadAll(r)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func makeRequest(handler http.Handler, method string, target string) *http.Response {
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)
	return w.Result()
}
