FROM mcr.microsoft.com/mssql-tools

COPY create_user.sql /usr/local/create_user.sql

CMD ["/opt/mssql-tools/bin/sqlcmd", "-S", "sql-server", "-U", "sa", "-P", "Password!", "-i", "/usr/local/create_user.sql"]
