package naming

import (
	"github.com/Xwudao/loom"

	"github.com/Xwudao/loom/internal/testdata/gen/naming/db"
	"github.com/Xwudao/loom/internal/testdata/gen/naming/provider"
	"github.com/Xwudao/loom/internal/testdata/gen/naming/web"
)

// mainAppGraph mirrors a real graph whose generated function is named after its
// target type. Both variables need a package-qualified name:
//
//   - db.DB would naturally be "db", which the db import already uses.
//   - web.MainApp would naturally be "mainApp", which is the function itself.
//
// The *settings.Settings value is produced by provider.NewSettings, so the
// settings package must not be imported at all.
var mainAppGraph = loom.Graph[*web.MainApp](
	loom.Name("mainApp"),
	loom.Provide(db.NewDB),
	loom.Provide(provider.NewSettings),
	loom.Provide(web.NewMainApp),
)
