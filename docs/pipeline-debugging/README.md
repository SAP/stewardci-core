# Pipeline Debugging Tips and Tricks

## Increasing the Log Level

Add this Groovy code to the pipeline script:

```groovy
import java.util.logging.Level
import java.util.logging.Logger
import java.util.logging.ConsoleHandler

Logger.getLogger("").setLevel(Level.FINEST)
for (h in Logger.getLogger("").getHandlers()) {
  if (h instanceof ConsoleHandler) {
    h.setLevel(Level.FINEST)
  }
}
```

You may choose log levels other than `FINEST`.
