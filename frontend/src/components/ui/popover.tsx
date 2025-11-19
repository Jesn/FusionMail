import * as React from "react"
import { cn } from "@/lib/utils"

type PopoverContextValue = {
  open: boolean
  setOpen: (open: boolean) => void
}

const PopoverContext = React.createContext<PopoverContextValue | undefined>(
  undefined
)

function usePopoverContext() {
  const context = React.useContext(PopoverContext)
  if (!context) {
    throw new Error("Popover components must be used within <Popover>")
  }
  return context
}

export interface PopoverProps {
  open?: boolean
  defaultOpen?: boolean
  onOpenChange?: (open: boolean) => void
  children: React.ReactNode
}

const Popover: React.FC<PopoverProps> = ({
  open: openProp,
  defaultOpen,
  onOpenChange,
  children,
}) => {
  const [uncontrolledOpen, setUncontrolledOpen] = React.useState(
    defaultOpen ?? false
  )

  const isControlled = openProp !== undefined
  const open = isControlled ? !!openProp : uncontrolledOpen

  const setOpen = React.useCallback(
    (value: boolean) => {
      if (!isControlled) {
        setUncontrolledOpen(value)
      }
      onOpenChange?.(value)
    },
    [isControlled, onOpenChange]
  )

  const value = React.useMemo(
    () => ({
      open,
      setOpen,
    }),
    [open, setOpen]
  )

  return (
    <PopoverContext.Provider value={value}>
      <div className="relative inline-block">{children}</div>
    </PopoverContext.Provider>
  )
}

Popover.displayName = "Popover"

export interface PopoverTriggerProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  asChild?: boolean
  children?: React.ReactNode
}

const PopoverTrigger = React.forwardRef<HTMLButtonElement, PopoverTriggerProps>(
  ({ asChild = false, children, ...props }, ref) => {
    const { open, setOpen } = usePopoverContext()

    const toggle = (event: React.MouseEvent<any>) => {
      if (!asChild && typeof props.onClick === "function") {
        props.onClick(event as React.MouseEvent<HTMLButtonElement>)
      }
      setOpen(!open)
    }

    if (asChild && React.isValidElement(children)) {
      const child = children as React.ReactElement<any>
      return React.cloneElement(child, {
        ref,
        onClick: (event: React.MouseEvent<any>) => {
          if (typeof child.props.onClick === "function") {
            child.props.onClick(event)
          }
          setOpen(!open)
        },
        "aria-expanded": open,
        "data-state": open ? "open" : "closed",
      })
    }

    return (
      <button
        type="button"
        ref={ref}
        aria-expanded={open}
        data-state={open ? "open" : "closed"}
        onClick={toggle}
        {...props}
      >
        {children}
      </button>
    )
  }
)

PopoverTrigger.displayName = "PopoverTrigger"

export interface PopoverContentProps
  extends React.HTMLAttributes<HTMLDivElement> {
  align?: "start" | "center" | "end"
}

const PopoverContent = React.forwardRef<HTMLDivElement, PopoverContentProps>(
  ({ className, align = "center", children, ...props }, ref) => {
    const { open, setOpen } = usePopoverContext()
    const contentRef = React.useRef<HTMLDivElement | null>(null)

    React.useImperativeHandle(ref, () => contentRef.current as HTMLDivElement)

    React.useEffect(() => {
      if (!open) return

      const handleKeyDown = (event: KeyboardEvent) => {
        if (event.key === "Escape") {
          setOpen(false)
        }
      }

      const handleClickOutside = (event: MouseEvent) => {
        if (!contentRef.current) return
        if (!contentRef.current.contains(event.target as Node)) {
          setOpen(false)
        }
      }

      document.addEventListener("keydown", handleKeyDown)
      document.addEventListener("mousedown", handleClickOutside)

      return () => {
        document.removeEventListener("keydown", handleKeyDown)
        document.removeEventListener("mousedown", handleClickOutside)
      }
    }, [open, setOpen])

    if (!open) {
      return null
    }

    const alignClass =
      align === "start"
        ? "left-0"
        : align === "end"
        ? "right-0"
        : "left-1/2 -translate-x-1/2"

    return (
      <div
        ref={contentRef}
        className={cn(
          "absolute z-50 mt-2 min-w-[8rem] rounded-md border bg-popover p-4 text-popover-foreground shadow-md",
          alignClass,
          className
        )}
        {...props}
      >
        {children}
      </div>
    )
  }
)

PopoverContent.displayName = "PopoverContent"

export { Popover, PopoverTrigger, PopoverContent }

