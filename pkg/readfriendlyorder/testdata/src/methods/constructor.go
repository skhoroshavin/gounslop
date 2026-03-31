package methods

type Handler struct{}

func (h *Handler) Handle() int { return 1 }

func NewHandler() *Handler { return &Handler{} } // want `Place constructor "NewHandler" right after type "Handler" declaration.`
