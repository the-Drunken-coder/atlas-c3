FROM python:3.12-slim
# The harness has no persistent volume, so we can simply create the unprivileged
# user, chown /app at build time, and drop to USER 10001 before CMD runs.
RUN groupadd --system --gid 10001 atlas \
 && useradd --system --no-create-home --shell /usr/sbin/nologin --gid 10001 --uid 10001 atlas \
 && mkdir -p /app \
 && chown -R atlas:atlas /app
WORKDIR /app
COPY --chown=atlas:atlas data-fusion /app/data-fusion
USER 10001:10001
CMD ["python", "/app/data-fusion/harness/runner/main.py"]
