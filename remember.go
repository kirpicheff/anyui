package anyui

import "git.gtw.su/kirpich/anyui/ui"

// Remember кеширует результат фоновой функции loader по ключу key для использования внутри drawFn.
// При первом вызове или после InvalidateRemember запускает loader в отдельной горутине
// и автоматически перерисовывает окно через App.Dispatch по завершении.
func Remember[T any](u *UI, key any, loader func() (T, error)) (val T, loading bool, err error) {
	if u == nil || u.ctx == nil || loader == nil {
		return val, false, nil
	}
	k := ui.RememberKey(key)
	if raw, cachedErr, isLoading, ok := u.ctx.RememberGet(k); ok {
		if isLoading {
			return val, true, nil
		}
		if typed, okCast := raw.(T); okCast {
			val = typed
		}
		return val, false, cachedErr
	}

	gen, shouldStart := u.ctx.RememberStart(k)
	if shouldStart {
		app := u.app
		ctx := u.ctx
		go func() {
			res, loadErr := loader()
			if app != nil {
				app.Dispatch(func() {
					ctx.RememberFinish(k, gen, res, loadErr)
				})
				return
			}
			ctx.RememberFinish(k, gen, res, loadErr)
		}()
	}
	return val, true, nil
}

// InvalidateRemember сбрасывает кеш Remember для указанного ключа и запрашивает перерисовку.
func (u *UI) InvalidateRemember(key any) {
	if u == nil || u.ctx == nil {
		return
	}
	u.ctx.InvalidateRemember(key)
}

// ClearRemember полностью очищает кеш Remember текущего окна и запрашивает перерисовку.
func (u *UI) ClearRemember() {
	if u == nil || u.ctx == nil {
		return
	}
	u.ctx.ClearRemember()
}

